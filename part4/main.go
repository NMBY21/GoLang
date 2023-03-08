package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Backend struct {
	net.Conn //has atype not remember name reader
	Reader *bufio.Reader
	Writer *bufio.Writer
}

var backendQueue chan *Backend
var requestBytes map[string]int64
var requestLock sync.Mutex

func init() {
	requestBytes = make(map[string]int64)
	backendQueue = make(chan *Backend, 10)
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen: %s", err)
	}
	log.Printf("Running proxy ")
	for {
		if conn, err := ln.Accept(); err == nil {
			go handleConnection(conn)
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			if err != io.EOF {
				log.Printf("Failed to read request: %s", err)
			}
			return
		}
		be, err := getBackend()
		if err != nil {
			return
		}
		if err := req.Write(be.Writer); err == nil {
			be.Writer.Flush()
			if resp, err := http.ReadResponse(be.Reader, req); err == nil {
				bytes := updateStats(req, resp)
				resp.Header.Set("X-Bytes", strconv.FormatInt(bytes, 10))
				FixHttp10Response(resp, req)
				if err := resp.Write(conn); err == nil {
					log.Printf("proxied %s: got %d", req.URL.Path, resp.StatusCode)
					bytes, _ := httputil.DumpRequest(req, false)
					log.Printf("Dump header: %s", string(bytes))
				}
				if resp.Close {
					return
				}
			}
		}
		go queueBackend(be)
	}
}

// updateStats takes a request and response and collects some statistics about them. This is
// very simple for now.
func updateStats(req *http.Request, resp *http.Response) int64 {
	requestLock.Lock()
	defer requestLock.Unlock()
	requestBytes[req.URL.Path] = resp.ContentLength
	return requestBytes[req.URL.Path]
}

// read from the backend channel
func getBackend() (*Backend, error) {
	select {
		// if the channel is empty, block
	case be := <-backendQueue:
		return be, nil
	case <-time.After(100 * time.Millisecond):

		be, err := net.Dial("tcp", "127.0.0.1:8081")
		if err != nil {
			return nil, err
		}
		return &Backend{
			Conn:   be,
			Reader: bufio.NewReader(be),
			Writer: bufio.NewWriter(be),
		}, nil
	}
}

// put to the backend queueBackend takes a backend and reenqueues it.
func queueBackend(be *Backend) {
	select {
	case backendQueue <- be:
		// Backend re-enqueued safely, move on.
	case <-time.After(1 * time.Second):
		// If this fires, it means the queue of backends was full. We don't want to wait
		// forever, as this period of time blocks us handling the next request a user
		// might send us.
		be.Close()
	}
}

// FixHttp10Response maybe downgrades an http.Response object to HTTP 1.0, which is necessary in
// the case where the original client request was 1.0.
func FixHttp10Response(resp *http.Response, req *http.Request) {
	if req.ProtoMajor == 1 && req.ProtoMinor == 1 {
		return
	}

	resp.Proto = "HTTP/1.0"
	resp.ProtoMajor = 1
	resp.ProtoMinor = 0

	if strings.Contains(strings.ToLower(req.Header.Get("Connection")), "keep-alive") {
		resp.Header.Set("Connection", "keep-alive")
		resp.Close = false
	} else {
		resp.Close = true
	}
}
