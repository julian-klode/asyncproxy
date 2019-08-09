# asyncproxy - Improve initial connection speed

[![Build Status](https://travis-ci.com/julian-klode/asyncproxy.svg?branch=master)](https://travis-ci.com/julian-klode/asyncproxy) [![GoDoc](https://godoc.org/github.com/julian-klode/asyncproxy?status.svg)](https://godoc.org/github.com/julian-klode/asyncproxy) [![Go Report Card](https://goreportcard.com/badge/github.com/julian-klode/asyncproxy)](https://goreportcard.com/report/github.com/julian-klode/asyncproxy)

asyncproxy is a http proxy that keeps connections in the background
for CONNECT requests done previously, so if a new CONNECT request
comes in for the same host, it can just use the established connection
and establish a new one for later use in the background.

This reduces the RTT for CONNECT requests on slow connections.
