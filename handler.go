package localstar

// Implements the plugin.Handler interface.

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// Name implements the plugin.Handler interface.
func (ls LocalStar) Name() string { return name }

// ServeDNS implements the plugin.Handler interface.
func (ls LocalStar) ServeDNS(ctx context.Context, w dns.ResponseWriter, req *dns.Msg) (int, error) {
	next := func() (int, error) {
		return plugin.NextOrFailure(ls.Name(), ls.next, ctx, w, req)
	}
	state := &request.Request{W: w, Req: req}
	if state.QClass() != dns.ClassINET {
		return next()
	}

	lookupName, err := ls.getLookupName(state.Name())
	if err != nil {
		return serveErrorCode(err)
	}

	rep, err := ls.lookupOnExternalDNS(ctx, lookupName, state.Name(), req)
	if err != nil {
		return serveErrorCode(err)
	}

	w.WriteMsg(rep)
	return dns.RcodeSuccess, nil
}

func serveErrorCode(err error) (int, error) {
	switch err {
	case errLoopRequest:
		return dns.RcodeRefused, err

	default:
		return dns.RcodeServerFailure, err
	}
}
