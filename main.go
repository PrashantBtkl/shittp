package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
)

var log = slog.New(slog.NewJSONHandler(os.Stdout, nil))

type HTTPRequest struct {
	Method  string
	URI     string
	Version string
	Headers map[string]string
	Body    []byte
}

type HTTPResponse struct {
	Version string
	Status  int
	Reason  string
	Headers map[string]string
	Body    []byte
}

func (r *HTTPRequest) parseRequest(line string) error {
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return fmt.Errorf("invalid request line: %s", line)
	}
	r.Method = parts[0]
	r.URI = parts[1]
	r.Version = parts[2]
	return nil
}

func (r *HTTPRequest) parseHeaders(lines []string) error {

	r.Headers = make(map[string]string)

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		r.Headers[key] = value
	}
	return nil
}

func (r *HTTPResponse) writeResponse(w *net.TCPConn) error {
	header := fmt.Sprintf("HTTP/%s %d %s\r\n", r.Version, r.Status, r.Reason)
	for key, value := range r.Headers {
		header += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	header += "\r\n"
	w.Write([]byte(header))
	w.Write(r.Body)
	return nil
}

func handleRequest(conn *net.TCPConn) {
	defer conn.Close()
	req := &HTTPRequest{}
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Error("failed to read request", "error", err.Error())
		handleFailure(conn)
		return
	}
	line := buf[:n]
	lines := strings.Split(string(line), "\r\n")

	err = req.parseRequest(lines[0])
	if err != nil {
		log.Error("failed to parse request", "error", err.Error())
		handleFailure(conn)
		return
	}
	headers := []string{}
	for _, line := range lines[1:] {
		headers = append(headers, line)
	}
	err = req.parseHeaders(headers)
	if err != nil {
		log.Error("failed to parse headers", "error", err.Error())
		handleFailure(conn)
		return
	}

	fmt.Printf("Request: %s %s %s\n", req.Method, req.URI, req.Version)
	fmt.Println("Headers:")
	for key, value := range req.Headers {
		fmt.Printf("  %s: %s\n", key, value)
	}

	resp := &HTTPResponse{
		Version: "1.1",
		Status:  200,
		Reason:  "OK",
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: []byte("Hello, World!"),
	}
	err = resp.writeResponse(conn)
	if err != nil {
		log.Error("failed to write response", "error", err.Error())
		return
	}
	log.Info("response served", "resp", resp)
}

func handleFailure(conn *net.TCPConn) {
	defer conn.Close()
	body := []byte("Internal Server Error")
	length := len(body)

	resp := &HTTPResponse{
		Version: "1.1",
		Status:  500,
		Reason:  "OK",
		Headers: map[string]string{
			"Content-Type":   "text/plain",
			"Content-Length": fmt.Sprint(length),
		},
		Body: body,
	}

	err := resp.writeResponse(conn)
	if err != nil {
		log.Error("failed to write response", "error", err.Error())
		return
	}
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Error("failed to create tcp connection", "error", err.Error())
		return
	}
	log.Info("Server listening on :8080")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error("failed to listen tcp connection", "error", err.Error())
			continue
		}
		go handleRequest(conn.(*net.TCPConn))
	}
}
