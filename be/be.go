// Simple HTTP server for evaluating AFE.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

var hostname = "localhost"
var port = 8080

func main() {
	rootHandler := func(w http.ResponseWriter, req *http.Request) {
		_, err := io.WriteString(w, "hello, world\n")
		if err != nil {
			log.Fatal(err)
		}
	}

	http.HandleFunc("/", rootHandler)
	log.Printf("HTTP server starting on %s:%d", hostname, port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", hostname, port), nil))
}
