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

func (coe connOrError) IsDead() bool {
	return time.Now().Sub(coe.time) >= timeOut
}

var timeOut = 5 * time.Second
var slots = make(map[string]chan connOrError)

var forceIPv4 = flag.Bool("4", false, "a bool")

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
				if tcpConn := conn.(*net.TCPConn); tcpConn != nil {
					if err := conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
						slots[protAndAddr] <- connOrError{nil, err, t}
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
		return coe.conn, coe.err
	}
}
