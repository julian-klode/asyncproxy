package main

import "flag"
import "fmt"
import "net"
import "log"
import "time"

type connOrError struct {
	conn net.Conn
	err  error
	time time.Time
}

var slots = make(map[string]chan connOrError)
var timeOutSec = flag.Int("timeout", 0, "timeout, in seconds")
var forceIPv4 = flag.Bool("4", false, "specify to force IPv4 connections to server")

func (coe connOrError) IsDead() bool {
	return *timeOutSec > 0 && time.Now().Sub(coe.time) >= time.Duration(*timeOutSec)*time.Second
}

// Dial dials a connection asynchronously, opening a new connection
// in the background once a connection has been taken. The connections
// use http.KeepAlive.
func Dial(network, addr string) (net.Conn, error) {
	if *forceIPv4 && (network == "tcp" || network == "tcp6") {
		network = "tcp4"
	}
	if *forceIPv4 && (network == "udp" || network == "udp") {
		network = "udp"
	}
	protAndAddr := fmt.Sprintf("%s,%s", network, addr)
	if slots[protAndAddr] == nil {
		slots[protAndAddr] = make(chan connOrError)
		go func() {
			for {
				t := time.Now()
				conn, err := net.Dial(network, addr)
				if conn != nil {
					if tcpConn := conn.(*net.TCPConn); tcpConn != nil {
						if err := conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
							slots[protAndAddr] <- connOrError{nil, err, t}
							continue
						}
					}
				}
				log.Printf("Finished %s dial to %s in %s", network, addr, time.Now().Sub(t))
				slots[protAndAddr] <- connOrError{conn, err, t}
			}
		}()
	}

	for {
		coe := <-slots[protAndAddr]
		if coe.IsDead() {
			log.Printf("Ignoring connection, timed out at age %s", time.Now().Sub(coe.time))
			continue
		}
		if coe.err != nil {
			log.Printf("Dial %s, %s: %s", network, addr, coe.err)
		}
		return coe.conn, coe.err
	}
}
