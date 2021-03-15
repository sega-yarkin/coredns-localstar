package main

import (
	_ "github.com/coredns/coredns/core/plugin"
	_ "github.com/sega-yarkin/coredns-localstar"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
)

func init() {
	// Insert localstart before whoami
	var to int
	for to = 0; to < len(dnsserver.Directives); to++ {
		if dnsserver.Directives[to] == "whoami" {
			break
		}
	}
	dnsserver.Directives = append(dnsserver.Directives, "")
	copy(dnsserver.Directives[to+1:], dnsserver.Directives[to:])
	dnsserver.Directives[to] = "localstar"
}

func main() {
	coremain.Run()
}
