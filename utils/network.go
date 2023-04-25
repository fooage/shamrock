package utils

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// This file is intended to adopt the special situation when there is no
// service discovery, and will be deprecated after the arch is perfected.

const (
	// port offset define port offset for multiple interface
	portOffsetHTTP = iota + 1
	portOffsetRPC
)

func AddressOffsetHTTP(url url.URL) string {
	host := strings.Split(url.Host, ":")[0]
	port, _ := strconv.Atoi(url.Port())
	port = port + portOffsetHTTP
	return fmt.Sprintf("%s:%d", host, port)
}

func AddressOffsetRPC(url url.URL) string {
	host := strings.Split(url.Host, ":")[0]
	port, _ := strconv.Atoi(url.Port())
	port = port + portOffsetRPC
	return fmt.Sprintf("%s:%d", host, port)
}
