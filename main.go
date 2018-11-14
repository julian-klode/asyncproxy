package main

import "flag"
import "net/http"

var listen = flag.String("listen", ":3129", "address to listen to")

func main() {
	flag.Parse()
	http.ListenAndServe(*listen, HTTPProxyHandler())
}
