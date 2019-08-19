package main

import (
	"afe/config"
	"flag"
	"io"
	"log"
	"net/http"
)

type Proxy struct {
	config config.ProxyConfig
}

var configPath = flag.String("config", "config.yaml", "full path to config file")

func main() {
	proxy := Proxy{}

	if err := config.ParseConfigFromFile(*configPath, &proxy.config); err != nil {
		log.Fatal(err)
	}

	log.Printf("config: %+v", proxy.config)

	http.HandleFunc("/", proxy.handler)
	log.Fatal(http.ListenAndServe(proxy.config.Listen.String(), nil))
}

func (proxy Proxy) handler(w http.ResponseWriter, req *http.Request) {
	_, err := io.WriteString(w, "hello, proxy\n")
	if err != nil {
		log.Fatal(err)
	}
}
