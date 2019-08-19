// Simple HTTP server for evaluating AFE.
//
// Listens on each service address in the config. Assumes that all the
// addresses and ports are listenable on the current host.
package main

import (
	"afe/config"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

var ProxyConfig config.ProxyConfig
var configpath = flag.String("config", "config.yaml", "full path to config file")

func main() {
	if err := config.ParseConfigFromFile(*configpath, &ProxyConfig); err != nil {
		log.Fatal(err)
	}

	log.Printf("%+v", ProxyConfig)

	var wg sync.WaitGroup

	for _, service := range ProxyConfig.Proxy.Services {
		for _, host := range service.Hosts {
			hostport := fmt.Sprintf("%s:%d", host.Address, host.Port)

			handler := func(w http.ResponseWriter, req *http.Request) {
				_, err := io.WriteString(w, fmt.Sprintf("service: %s, addr: %s",
					service.Name, hostport))
				if err != nil {
					log.Fatal(err)
				}
			}

			wg.Add(1)

			s := &http.Server{
				Addr:    hostport,
				Handler: http.HandlerFunc(handler),
			}

			log.Printf("HTTP server starting on %s\n", hostport)
			go func() {
				log.Fatal(s.ListenAndServe())
				wg.Done() // NOTREACHED
			}()
		}
	}

	wg.Wait()
}
