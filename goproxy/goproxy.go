package goproxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

type ProxyHandler struct{}

func NewHandler() *ProxyHandler {
	return &ProxyHandler{}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var status int
	var err error

	switch req.Method {
	case http.MethodConnect:
		status, err = h.proxyConnect(w, req)
	case http.MethodGet:
		status, err = h.proxyGet(w, req)
	default:
		status, err = http.StatusBadRequest, fmt.Errorf("bad request:%s", req.Method)
	}

	if status != 0 {
		w.WriteHeader(status)
		log.Println("error proxying request:", err.Error())
		return
	}

	log.Println("completed proxying", req.URL.Host)
}

func (h *ProxyHandler) proxyGet(w http.ResponseWriter, req *http.Request) (int, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Println("proxying GET:", err)
		return http.StatusBadGateway, fmt.Errorf("rountripping:%w", err)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Println("error closing req body", err.Error())
		}
	}()

	for k, v := range resp.Header {
		w.Header().Set(k, strings.Join(v, ", "))
	}
	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed copy of body into response")
	}
	log.Println("done proxying GET")
	return 0, nil
}

func (h *ProxyHandler) proxyConnect(w http.ResponseWriter, req *http.Request) (int, error) {
	log.Println("new", req.Method, "to", req.URL.Host)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return http.StatusServiceUnavailable, fmt.Errorf("unable to hijack connection")
	}

	log.Println("dialing")
	outbound, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("failed to connect to %s: %w", req.URL.Host, err)
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
		return http.StatusInternalServerError, fmt.Errorf("failed to hijack connection:%w", err)
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
		return 0, fmt.Errorf("failed to copy outbound to conn: %w", err)
	}

	return 0, nil
}
