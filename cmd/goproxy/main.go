package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/Aldo-Cassola/goproxy/goproxy"
)

var swVersion = "undefined"

func main() {
	log.SetPrefix("GoProxy v" + swVersion + ":")
	listenAddr := flag.String("listen", "127.0.0.1:8080", "address:port to listen on")

	srv := &http.Server{
		Handler: goproxy.NewHandler(),
		Addr:    *listenAddr,
	}

	log.Println("listening on", *listenAddr)
	log.Println(srv.ListenAndServe())
}
