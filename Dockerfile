FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .

ENV CGO_ENABLED=0
RUN go build -o http_server http_server.go
RUN go build -o proxy proxy.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/http_server .
COPY --from=builder /app/proxy .

COPY index.html .
COPY newfile.txt . 


EXPOSE 8080

EXPOSE 9090


CMD ["./http_server", "8080"]