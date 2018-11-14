package main

import "flag"
import "net/http"

func main() {
	flag.Parse()
	http.ListenAndServe(":3129", HTTPProxyHandler())
}
