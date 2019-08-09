package main

import "flag"
import "fmt"
import "net"
import "log"
import "sync"
import "time"

type connOrError struct {
	conn net.Conn
	err  error
	time time.Time
}

// AsyncDialer provides a Dial() method that pre-dials asynchronously.
// Use NewAsyncDialer() to create a new dialer.
type AsyncDialer struct {
	slots map[string]chan connOrError
	mutex sync.Mutex
}

var timeOutSec = flag.Int("timeout", 0, "timeout, in seconds")
var forceIPv4 = flag.Bool("4", false, "specify to force IPv4 connections to server")

func (coe connOrError) IsDead() bool {
	return *timeOutSec > 0 && time.Now().Sub(coe.time) >= time.Duration(*timeOutSec)*time.Second
}

// NewAsyncDialer creates a new AsyncDialer
func NewAsyncDialer() *AsyncDialer {
	return &AsyncDialer{
		slots: make(map[string]chan connOrError),
	}
}

// Dial dials a connection asynchronously, opening a new connection
// in the background once a connection has been taken. The connections
// use http.KeepAlive.
func (dialer *AsyncDialer) Dial(network, addr string) (net.Conn, error) {
	if *forceIPv4 && (network == "tcp" || network == "tcp6") {
		network = "tcp4"
	}
	if *forceIPv4 && (network == "udp" || network == "udp6") {
		network = "udp4"
	}
	protAndAddr := fmt.Sprintf("%s,%s", network, addr)
	dialer.mutex.Lock()
	if dialer.slots[protAndAddr] == nil {
		dialer.slots[protAndAddr] = make(chan connOrError)
		go func() {
			for {
				t := time.Now()
				conn, err := net.Dial(network, addr)
				if conn != nil {
					if tcpConn := conn.(*net.TCPConn); tcpConn != nil {
						if err := conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
							dialer.slots[protAndAddr] <- connOrError{nil, err, t}
							continue
						}
					}
				}
				log.Printf("Finished %s dial to %s in %s", network, addr, time.Now().Sub(t))
				dialer.slots[protAndAddr] <- connOrError{conn, err, t}
			}
		}()
	}
	dialer.mutex.Unlock()

	for {
		coe := <-dialer.slots[protAndAddr]
		if coe.IsDead() {
			log.Printf("Ignoring connection, timed out at age %s", time.Now().Sub(coe.time))
			if coe.conn != nil {
				coe.conn.Close()
			}
			continue
		}
		if coe.err != nil {
			log.Printf("Dial %s, %s: %s", network, addr, coe.err)
		}
		return coe.conn, coe.err
	}
}
