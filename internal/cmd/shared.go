package cmd

import (
	"path"
	"runtime"
)

func loopback() string {
	switch runtime.GOOS {
	case "darwin":
		return "lo0"
	default:
		return "lo"
	}
}

var discoveryAddr = path.Join("/ip4/228.8.8.8/udp/8822/multicast", loopback())

func BootstrapAddr() string {
	return discoveryAddr
}
