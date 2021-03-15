package localstar

import (
	"strconv"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/parse"
	"github.com/miekg/dns"
)

func init() {
	plugin.Register(name, setup)
}

func setup(cc *caddy.Controller) error {
	ls := newLocalStar(dnsserver.GetConfig(cc))
	err := parseConfig(cc, &ls)
	if err != nil {
		return plugin.Error(name, err)
	}

	dnsserver.GetConfig(cc).AddPlugin(func(next plugin.Handler) plugin.Handler {
		ls.next = next
		return ls
	})
	return nil
}

func parseConfig(cc *caddy.Controller, ls *LocalStar) error {
	var err error
	for err == nil && cc.Next() {
		for err == nil && cc.NextBlock() {
			switch cc.Val() {
			default:
				err = cc.Errf("unknown property: '%s'", cc.Val())
			case "to_zone":
				err = parseConfigToZone(cc, ls)
			case "endpoint", "endpoints":
				err = parseConfigEndpoints(cc, ls)
			case "prefix_len":
				err = parseConfigPrefixLen(cc, ls)
			case "timeout":
				err = parseConfigTimeout(cc, ls)
			}

			if len(cc.RemainingArgs()) > 0 {
				err = cc.ArgErr()
			}
		}
	}
	if err != nil {
		return err
	}

	if ls.toZone == "" {
		return cc.Err("'to_zone' parameter is required")
	}

	if len(ls.endpoints) < 1 {
		ls.endpoints, err = parse.HostPortOrFile(ls.defaultEndpoints...)
		if err != nil || len(ls.endpoints) < 1 {
			return cc.Err("no endpoints specified")
		}
	}

	ls.provider = new(simpleDNSProvider)
	if err = ls.provider.Init(ls.endpoints, ls.timeout); err != nil {
		return cc.Errf("cannot init provider: %s", err.Error())
	}
	return nil
}


func parseConfigToZone(cc *caddy.Controller, ls *LocalStar) error {
	if !cc.NextArg() {
		return cc.ArgErr()
	}
	ls.toZone = dns.CanonicalName(cc.Val())
	if dns.IsSubDomain(ls.fromZone, ls.toZone) {
		return cc.Err("'to_zone' cannot be equal to or be a child of serving zone")
	}
	ls.toZoneDiff = calcZoneDiff(ls.fromZone, ls.toZone)
	return nil
}

func parseConfigEndpoints(cc *caddy.Controller, ls *LocalStar) error {
	endpoints := cc.RemainingArgs()
	if len(endpoints) == 0 {
		return cc.ArgErr()
	}
	endpoints, err := parse.HostPortOrFile(endpoints...)
	if err != nil {
		return err
	}
	ls.endpoints = append(ls.endpoints, endpoints...)
	return nil
}

func parseConfigPrefixLen(cc *caddy.Controller, ls *LocalStar) (err error) {
	if !cc.NextArg() {
		return cc.ArgErr()
	}
	ls.prefixLen, err = strconv.Atoi(cc.Val())
	if err != nil {
		return cc.Errf("invalid number: %q", cc.Val())
	}
	if ls.prefixLen < 1 {
		return cc.Errf("prefix_len can't be less than 1: %d", ls.prefixLen)
	}
	return nil
}

func parseConfigTimeout(cc *caddy.Controller, ls *LocalStar) (err error) {
	if !cc.NextArg() {
		return cc.ArgErr()
	}
	ls.timeout, err = time.ParseDuration(cc.Val())
	if err != nil {
		return cc.Errf("invalid duration: %q", cc.Val())
	}
	if ls.timeout < 0 {
		return cc.Errf("timeout can't be negative: %d", ls.timeout)
	}
	if ls.timeout == 0 {
		ls.timeout = defaultTimeout
	}
	return nil
}
