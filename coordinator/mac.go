package main

import (
	"crypto/rand"
	"fmt"
)

var buf [6]byte

func generateMACAddress() string {
	rand.Read(buf[:])
	// unicast
	buf[0] &^= (1 << 0)
	// local
	buf[0] |= (1 << 1)
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[01], buf[2], buf[3], buf[4], buf[5])
}
