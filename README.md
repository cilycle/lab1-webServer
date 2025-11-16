# Lab 1: HTTP Server

## 1. Core Features

### `http_server` (The Server)
* **Concurrency Model:** Spawns a new goroutine for each connection. Uses a **buffered channel (semaphore)** to limit the maximum number of concurrent connections to **10**.
* **`GET` Method:** Supports serving files with correct `Content-Type` mapping for `.html`, `.txt`, `.css`, `.jpg`, `.jpeg`, and `.gif`.
* **`POST` Method:** Supports receiving data from a client's request body and saving it as a local file on the server.
* **Error Handling:**
    * `404 Not Found`: For requests for non-existent files.
    * `400 Bad Request`: For unsupported file types or malformed requests.
    * `501 Not Implemented`: For all methods other than `GET` and `POST` (e.g., `PUT`, `DELETE`).

### `proxy` (The Proxy)
* **`GET` Method:** Implements `GET` request forwarding. It connects to the origin server, forwards the client's request, and streams the origin server's full response (headers and body) back to the client.
* **Error Handling:**
    * `501 Not Implemented`: For all methods other than `GET`.

## 2. How to Run (Docker - Recommended Method)

This is the recommended way to run the project for a demo or grading.

### Prerequisites
1.  **Docker Desktop** must be installed and running.
2.  Create a test `index.html` file in the project root (`D:\go-webserver`) for testing `GET` requests:
    ```powershell
    # Run in PowerShell:
    Set-Content -Path "index.html" -Value "<html><body>Hello from Docker!</body></html>"
    ```

### Step 1: Build the Docker Image

In your PowerShell terminal, `cd` to the project directory and run:

```powershell
# -t my-go-server gives the image a name (t = tag)
# . (dot) tells Docker to find the Dockerfile in the current directory
docker build -t go-webserver .
```

### Step 2: Run the Containers
#### 1. Start the http_server on port 8080
```powershell
docker run -d -p 8080:8080 --name http-server go-webserver
```

#### 2. Start the proxy on port 9090
```powershell
#    We override the default CMD in the Dockerfile to run the proxy
docker run -d -p 9090:9090 --name http-proxy go-webserver ./proxy 9090
```

#### 3. Server test
```powershell
curl -Method GET http://localhost:8080/index.html
```

#### 4. Proxy test
```powershell
curl -Method GET http://localhost:8080/index.html -Proxy http://localhost:9090
```
