package main

import "net"
import "log"
import "time"

type connOrError struct {
	conn net.Conn
	err error
}

var slots = make(map[string]chan connOrError)

// Dial dials a connection asynchronously, opening a new connection
// in the background once a connection has been taken. The connections
// use http.KeepAlive.
func Dial(network, addr string) (net.Conn, error) {
	if slots[addr] == nil {
		slots[addr] = make(chan connOrError)
		go func() {
			for {
				t := time.Now()
				conn, err := net.Dial(network, addr)
				if tcpConn := conn.(*net.TCPConn); tcpConn != nil {
					if err := conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
						slots[addr] <- connOrError{nil, err}
					}
				}
				elapsed := time.Now().Sub(t)
				log.Printf("Finished %s dial to %s in %s", network, addr, elapsed)
				if err != nil {
					slots[addr] <- connOrError{nil, err}
				} else {
					slots[addr] <- connOrError{conn, nil}
				}
			}
		}()
	}

	coe := <-slots[addr]
	return coe.conn, coe.err
}
