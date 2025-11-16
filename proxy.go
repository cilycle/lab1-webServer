package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

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
	log.Printf("Proxy will start on %s...", address)
	// step 2: Listen on the port
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	// step 3: Accept connections loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// step 4: Start a goroutine for each connection
		go handleProxyRequest(conn)
	}
}

func handleProxyRequest(clientConn net.Conn) {
	defer clientConn.Close()
	log.Printf("Handling new proxy connection: %s", clientConn.RemoteAddr().String())

	reader := bufio.NewReader(clientConn)

	// step 1: Parse request
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Failed to parse request: %v", err)
		if err != io.EOF && !strings.Contains(err.Error(), "connection reset") {
			sendErrorResponse(clientConn, http.StatusBadRequest, "Bad Request")
		}
		return
	}

	// step 2: Only implement GET method
	if req.Method != "GET" {
		log.Printf("Unsupported method: %s", req.Method)
		sendErrorResponse(clientConn, http.StatusNotImplemented, "Not Implemented")
		return
	}

	log.Printf("Proxying %s %s", req.Method, req.URL.String())

	// step 3: Forward request to target server
	forwardRequest(clientConn, req)
}

func forwardRequest(clientConn net.Conn, req *http.Request) {
	// step 1: Get target host address
	targetHost := req.URL.Host
	if targetHost == "" {
		// If URL is a relative path (non-standard proxy request), try to get from Host header
		targetHost = req.Host
	}
	if targetHost == "" {
		sendErrorResponse(clientConn, http.StatusBadRequest, "Bad Request: Missing host in request")
		return
	}

	// step 2: Ensure target address includes port (default 80 for HTTP)
	if _, _, err := net.SplitHostPort(targetHost); err != nil {
		// Assume no port, add default 80 port
		targetHost = net.JoinHostPort(targetHost, "80")
	}

	// step 3: Connect to target server
	remoteConn, err := net.Dial("tcp", targetHost)
	if err != nil {
		log.Printf("Failed to connect to target server %s: %v", targetHost, err)
		sendErrorResponse(clientConn, http.StatusBadGateway, "Bad Gateway: Could not connect to host")
		return
	}
	defer remoteConn.Close()

	// step 4: Forward client request to target server

	req.RequestURI = req.URL.Path

	// Remove proxy-specific headers
	req.Header.Del("Proxy-Connection")
	req.Header.Set("Connection", "close") // Force close connection to simplify handling

	if err := req.Write(remoteConn); err != nil {
		log.Printf("Failed to forward request to %s: %v", targetHost, err)
		sendErrorResponse(clientConn, http.StatusBadGateway, "Bad Gateway: Error writing to remote")
		return
	}

	// step 5: Copy the target server's response *as is* back to the client
	// io.Copy copies status line, all headers, and body
	bytesCopied, err := io.Copy(clientConn, remoteConn)
	if err != nil {
		log.Printf("Failed to copy response from %s: %v", targetHost, err)
	}
	log.Printf("Copied %d bytes of response from %s", bytesCopied, targetHost)
}

// sendErrorResponse is a helper function to send error responses (same as server version)
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