package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type connOrError struct {
	conn    net.Conn
	err     error
	timeout time.Time
}

// AsyncDialer provides a Dial() method that pre-dials asynchronously.
type AsyncDialer struct {
	// TimeOutSec specifies a timeout for TCP operations
	TimeOutSec int
	// ForceIPv4 allows overriding the Dial request to enforce IPv4
	ForceIPv4 bool

	slots map[string]chan connOrError
	mutex sync.Mutex
}

func (coe connOrError) IsDead() bool {
	return coe.timeout.Sub(time.Now()) < 0
}

func (dialer *AsyncDialer) getChannel(network, addr string) chan connOrError {
	protAndAddr := fmt.Sprintf("%s,%s", network, addr)

	dialer.mutex.Lock()
	if dialer.slots == nil {
		dialer.slots = make(map[string]chan connOrError)
	}
	if dialer.slots[protAndAddr] == nil {
		dialer.slots[protAndAddr] = make(chan connOrError)
		go dialer.backgroundDialLoop(dialer.slots[protAndAddr], addr, network)
	}
	dialer.mutex.Unlock()

	return dialer.slots[protAndAddr]
}

// backgroundDialLoop dials in the background and adds the new connection (or error) to the specified channel.
func (dialer *AsyncDialer) backgroundDialLoop(channel chan connOrError, addr string, network string) {
	for {
		t := time.Now()
		timeout := t.Add(time.Duration(dialer.TimeOutSec) * time.Second)
		conn, err := net.Dial(network, addr)
		if conn != nil {
			if tcpConn := conn.(*net.TCPConn); tcpConn != nil {
				if err := conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
					conn.Close()
					channel <- connOrError{nil, err, timeout}
					continue
				}
			}
		}
		log.Printf("Finished %s dial to %s in %s", network, addr, time.Now().Sub(t))
		channel <- connOrError{conn, err, timeout}
	}
}

// Dial dials a connection asynchronously, opening a new connection
// in the background once a connection has been taken. The connections
// use http.KeepAlive.
func (dialer *AsyncDialer) Dial(network, addr string) (net.Conn, error) {
	if dialer.ForceIPv4 && (network == "tcp" || network == "tcp6") {
		network = "tcp4"
	}
	if dialer.ForceIPv4 && (network == "udp" || network == "udp6") {
		network = "udp4"
	}

	for connOrErr := range dialer.getChannel(network, addr) {
		if connOrErr.IsDead() {
			log.Printf("Ignoring connection, timed out")
			if connOrErr.conn != nil {
				connOrErr.conn.Close()
			}
			continue
		}
		if connOrErr.err != nil {
			log.Printf("Dial %s, %s: %s", network, addr, connOrErr.err)
		}
		return connOrErr.conn, connOrErr.err
	}

	panic(fmt.Sprintf("Somebody closed the channel for %s:%s", network, addr))
}
