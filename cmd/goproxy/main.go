package main

import (
	"io"
	"log"
	"net"
	"net/http"
)

func main() {
	srv := &http.Server{
		Handler: &proxyHandler{},
		Addr:    "127.0.0.1:8080",
	}

	log.Println(srv.ListenAndServe())
}

type proxyHandler struct{}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodConnect {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("bad request")
		return
	}

	log.Println("new CONNECT to", req.URL.Host)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		w.WriteHeader(http.StatusServiceUnavailable)
		log.Println("unable to hijack connection")
		return
	}

	log.Println("dialing")
	outbound, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		log.Println("failed to connect to", req.URL.Host+":", err)
		return
	}
	defer func() {
		err := outbound.Close()
		if err != nil {
			log.Println("failed closing outbound:", err.Error())
		}
	}()
	w.WriteHeader(http.StatusOK)

	inbound, buf, err := hijacker.Hijack()
	if err != nil {
		log.Println("failed to hijack connection:", err)
		return
	}

	defer func() {
		if err := inbound.Close(); err != nil {
			log.Println("failed to close inbound connection:", err.Error())
		}
	}()
	go func() {
		log.Println("copying buf")
		_, err = io.Copy(outbound, buf)
		if err != nil {
			log.Println("failed to copy buffer to outbound:", err)
		}

		if _, err := io.Copy(outbound, inbound); err != nil {
			log.Println("failed to copy connection to outbound", err)
		}
	}()

	log.Println("copying outbound to conn")
	if _, err := io.Copy(inbound, outbound); err != nil {
		log.Println("failed to copy outbound to conn", err)
		return
	}

	log.Println("completed CONNECT to", req.URL.Host)
}
