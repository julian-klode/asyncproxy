package main

import "flag"
import "net/http"

func main() {
	listen := flag.String("listen", ":3129", "address to listen to")
	timeOutSec := flag.Int("timeout", 60, "timeout, in seconds")
	forceIPv4 := flag.Bool("4", false, "specify to force IPv4 connections to server")
	flag.Parse()

	dialer := AsyncDialer{
		TimeOutSec: *timeOutSec,
		ForceIPv4:  *forceIPv4,
	}
	http.ListenAndServe(*listen, HTTPProxyHandler(&dialer))
}
