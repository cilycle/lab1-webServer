package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// define the maximum number of concurrent requests
const maxConcurrentRequests = 10

// Supported MIME types
var mimeTypes = map[string]string{
	".html": "text/html",
	".txt":  "text/plain",
	".gif":  "image/gif",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".css":  "text/css",
}

func main() {
	// step 1: Check and get command line argument (port)
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <port>", os.Args[0])
	}
	port := os.Args[1]
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("Invalid port: %s", port)
	}
	address := ":" + port
	log.Printf("Server will start on %s...", address)

	// step 2: Listen on the port
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", address, err)
	}
	defer listener.Close()

	// step 3: Limit concurrent requests
	sem := make(chan struct{}, maxConcurrentRequests)

	// step 4: Accept connections loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		sem <- struct{}{}
		// step 5: Start a goroutine for each connection
		go handleConnection(conn, sem)
	}
}

func handleConnection(conn net.Conn, sem chan struct{}) {
	// Ensure the connection is closed and semaphore is released when the function exits
	defer conn.Close()
	defer func() {
		<-sem // Release semaphore
		log.Printf("Connection %s closed, released a slot", conn.RemoteAddr().String())
	}()

	log.Printf("Handling new connection: %s", conn.RemoteAddr().String())
	reader := bufio.NewReader(conn)

	// step 1: Parse request (using net/http parser)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Failed to parse request: %v", err)
		if err != io.EOF && !strings.Contains(err.Error(), "connection reset") {
			sendErrorResponse(conn, http.StatusBadRequest, "Bad Request")
		}
		return
	}

	// step 2: Route based on method
	switch req.Method {
	case "GET":
		handleGet(conn, req)
	case "POST":
		handlePost(conn, req)
	default:
		// Other methods return 501 Not Implemented
		sendErrorResponse(conn, http.StatusNotImplemented, "Not Implemented")
	}
}

func handleGet(conn net.Conn, req *http.Request) {
	path := filepath.Clean("./" + req.URL.Path)
	if path == "./" {
		path = "./index.html" // Default to serving index.html
	}

	// step 1: Check extension and Content-Type
	ext := filepath.Ext(path)
	contentType, ok := mimeTypes[ext]
	if !ok {
		log.Printf("Unsupported file type: %s (path: %s)", ext, path)
		sendErrorResponse(conn, http.StatusBadRequest, "Bad Request: Unsupported file type")
		return
	}

	// step 2: Try to open the file
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("File not found: %s", path)
			sendErrorResponse(conn, http.StatusNotFound, "Not Found")
		} else {
			log.Printf("Failed to open file: %v", err)
			sendErrorResponse(conn, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}
	defer file.Close()

	// step 3: Get file size (for Content-Length)
	stat, err := file.Stat()
	if err != nil {
		log.Printf("Failed to get file stat: %v", err)
		sendErrorResponse(conn, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	fileSize := stat.Size()

	// step 4: Send 200 OK response headers
	fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\n")
	fmt.Fprintf(conn, "Content-Type: %s\r\n", contentType)
	fmt.Fprintf(conn, "Content-Length: %d\r\n", fileSize)
	fmt.Fprintf(conn, "Connection: close\r\n") 
	fmt.Fprintf(conn, "\r\n") // End of headers

	// step 5: Send file content (body)
	_, err = io.Copy(conn, file)
	if err != nil {
		log.Printf("Failed to send file body: %v", err)
	}
}

func handlePost(conn net.Conn, req *http.Request) {
	// step 1: Similarly clean the path
	path := filepath.Clean("./" + req.URL.Path)

	// step 2: Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Failed to create directory: %v", err)
		sendErrorResponse(conn, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// step 3: Create file (overwrite if exists)
	file, err := os.Create(path)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		sendErrorResponse(conn, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer file.Close()

	// step 4: Write request body (req.Body) to file
	bytesCopied, err := io.Copy(file, req.Body)
	if err != nil {
		log.Printf("Failed to write to file: %v", err)
		sendErrorResponse(conn, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	log.Printf("Successfully POSTed %d bytes to %s", bytesCopied, path)

	// step 5: Send 201 Created response
	fmt.Fprintf(conn, "HTTP/1.1 201 Created\r\n")
	fmt.Fprintf(conn, "Content-Type: text/plain\r\n")
	fmt.Fprintf(conn, "Content-Length: 0\r\n")
	fmt.Fprintf(conn, "Connection: close\r\n")
	fmt.Fprintf(conn, "\r\n")
}

// sendErrorResponse is a helper function to send error responses
func sendErrorResponse(conn net.Conn, code int, status string) {
	body := fmt.Sprintf("%d %s", code, status)
	log.Printf("Sending error: %s", body)

	fmt.Fprintf(conn, "HTTP/1.1 %d %s\r\n", code, status)
	fmt.Fprintf(conn, "Content-Type: text/plain\r\n")
	fmt.Fprintf(conn, "Content-Length: %d\r\n", len(body))
	fmt.Fprintf(conn, "Connection: close\r\n")
	fmt.Fprintf(conn, "\r\n") // End of headers
	fmt.Fprintf(conn, "%s", body)
}