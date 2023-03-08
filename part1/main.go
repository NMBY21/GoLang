package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
)

func main() {
	// 1. Listen for incoming connections./socket/infinite loop
	if ln, err := net.Listen("tcp",":8080"); err == nil {
		for {
			// 2. Accept connection
			if conn, err := ln.Accept(); err == nil {
				reader := bufio.NewReader(conn)

				// 3. Read requests from the client
				if req, err := http.ReadRequest(reader); err == nil {

					//4. talking to backends.
					if be, err := net.Dial("tcp", "127.0.0.1:8081"); err == nil {
						be_reader := bufio.NewReader(be)

						// 5. proxying request to be.send the request to backend.
						if err := req.Write(be); err == nil {

							// 6. read the response from the backend.

							if resp, err := http.ReadResponse(be_reader, req); err == nil {

								// 7. Send the reponse to the client
								resp.Close = true
								
								// check whether the rwsponse is work 
								if err := resp.Write(conn); err == nil {

									log.Printf("%s: %d", req.URL.Path, resp.StatusCode)

								}

								conn.Close()
							}
						}
					}
				}
			}
		}
	}

}
