package main

import "net"
import "log"
import "time"

var slots map[string]chan net.Conn = make(map[string]chan net.Conn)

func Dial(network, addr string) (net.Conn, error) {
	if slots[addr] == nil {
		slots[addr] = make(chan net.Conn)
		go func() {
			for {
				t := time.Now()
				//log.Printf("Starting %s dial to %s", network, addr)
				conn, err := net.Dial(network, addr)
				elapsed := time.Now().Sub(t)
				log.Printf("Finished %s dial to %s in %s", network, addr, elapsed)
				slots[addr] <- conn
				//log.Printf("Sent %s dial to %s", network, addr)
				if err != nil {
					log.Printf("Error: %s", err)
					break
				}
			}
		}()
	}

	return <-slots[addr], nil

}
