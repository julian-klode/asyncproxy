package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// copyAndClose copies bytes from src to dst and closes both afterwards
func copyAndClose(dst io.WriteCloser, src io.ReadCloser) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Println("Could not forward:", err)
	}
	src.Close()
	dst.Close()
}

// httpProxyHandler implements a http.Handler for proxying requests
type httpProxyHandler struct {
	client http.Client
}

// serveHTTPConnect serves proxy requests for the CONNECT method. It does not
// print errors, but rather returns them for your proxy handler to handle.
func (proxy *httpProxyHandler) serveHTTPConnect(w http.ResponseWriter, r *http.Request) error {
	t := time.Now()
	//log.Println("Dialing for CONNECT to", r.URL.Host)
	remote, err := Dial("tcp", r.URL.Host)
	//log.Println("Got remote", remote, "err", err)
	if err != nil {
		w.WriteHeader(503)
		return err
	}
	w.WriteHeader(200)

	conn, _, err := w.(http.Hijacker).Hijack()
	log.Println("Hijacked conn", conn, "err", err, "in", time.Now().Sub(t))
	if err != nil {
		return err
	}

	go copyAndClose(remote, conn)
	copyAndClose(conn, remote)

	return nil
}

// ServeHTTP serves proxy requests
func (proxy *httpProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// net/http.Client does not handle the CONNECT stuff that well below, so
	// let us go a more direct route here - this could be used for the other
	// methods as well, but that would prevent reusing connections to the
	// proxy.
	if r.Method == "CONNECT" {
		if err := proxy.serveHTTPConnect(w, r); err != nil {
			log.Println(err)
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		}
		return
	}

	// The wonderful GET/POST/PUT/HEAD wonderland - this actually uses the
	// http library with a fake dial function that allows us to cache and
	// reuse connections to the proxy, speeding up the whole affair quite
	// a bit if you have to do TLS handshakes.
	if r.URL.Scheme == "" {
		r.URL.Scheme = "http"
		r.URL.Host = r.Host
	}
	r.RequestURI = ""
	res, err := proxy.client.Do(r)
	if err != nil {
		log.Println("Could not do", r, "-", err)
		w.WriteHeader(500)
		return
	}

	for k, vs := range res.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
	res.Body.Close()
}

// HTTPProxyHandler constructs a handler for http.ListenAndServe()
// that proxies HTTP requests via the configured proxies. It supports
// not only HTTP proxy requests, but also normal HTTP/1.1 requests with a
// Host header - thus enabling the use as a transparent proxy.
func HTTPProxyHandler() http.Handler {

	log.Printf("Forwarding HTTP")

	transport := http.Transport{
		MaxIdleConns:        64,
		MaxIdleConnsPerHost: 64,
		IdleConnTimeout:     5 * time.Minute,

		Dial: func(network, addr string) (net.Conn, error) {
			return Dial(network, addr)
		},
	}
	client := http.Client{
		Transport: &transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &httpProxyHandler{client}
}
