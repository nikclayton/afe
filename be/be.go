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
			wg.Add(1)

			hostport := host.String() // Avoid capturing host variable in go func()
			go func() {
				log.Printf("HTTP server starting on %s", hostport)

				handler := func(w http.ResponseWriter, req *http.Request) {
					log.Printf("%s handling request for %s", hostport, req.URL)
					_, err := io.WriteString(w, fmt.Sprintf("service: %s, addr: %s",
						service.Name, hostport))
					if err != nil {
						log.Fatal(err)
					}
				}

				log.Fatal(http.ListenAndServe(hostport, http.HandlerFunc(handler)))
				wg.Done() // NOTREACHED
			}()
		}
	}

	wg.Wait()
}
