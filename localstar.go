package localstar

import (
	"context"
	"errors"
	"strings"
	"time"

	// clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

const (
	name = "localstar"
	defaultTimeout = 5 * time.Second
)
// var log = clog.NewWithPlugin(name)

var (
	errLoopRequest = errors.New("loop request")
)

// LocalStar is a plugin that forward requests to other zone.
type LocalStar struct {
	fromZone string
	toZone string
	toZoneDiff string
	endpoints []string
	prefixLen int // in labels
	timeout time.Duration
	defaultEndpoints []string
	provider dnsProvider
	next plugin.Handler
}

func newLocalStar(config *dnsserver.Config) LocalStar {
	return LocalStar{
		fromZone: config.Zone,
		endpoints: []string{},
		prefixLen: 1,
		timeout: defaultTimeout,
		defaultEndpoints: []string{"/etc/resolv.conf"},
	}
}

func calcZoneDiff(from, to string) string {
	if strings.HasSuffix(from, "." + to) {
		return from[:len(from)-len(to)]
	}
	return ""
}

func (ls LocalStar) getLookupName(qname string) (string, error) {
	prefix := qname[:len(qname)-len(ls.fromZone)]
	if ls.toZoneDiff != "" && strings.HasSuffix(prefix, ls.toZoneDiff) {
		return "", errLoopRequest
	}
	parts := dns.SplitDomainName(prefix)
	cnt := ls.prefixLen
	if cnt > len(parts) {
		cnt = len(parts)
	}
	prefix = strings.Join(parts[len(parts)-cnt:], ".")
	return prefix + "." + ls.toZone, nil
}

func copyMsgWithQName(src *dns.Msg, name string) *dns.Msg {
	dst := src.Copy()
	dst.Question[0].Name = dns.Fqdn(name)
	return dst
}

func replaceRRName(rrs []dns.RR, from, to string) {
	for _, rr := range rrs {
		if rr.Header().Name == from {
			rr.Header().Name = to
		}
	}
}

func (ls LocalStar) lookupOnExternalDNS(
	ctx context.Context,
	lookupName string,
	origName string,
	req *dns.Msg,
) (*dns.Msg, error) {
	msg := copyMsgWithQName(req, lookupName)
	res, err := ls.provider.Exchange(ctx, msg)
	if err != nil {
		return nil, err
	}

	res.SetReply(req)
	res.Authoritative = false
	// r.RecursionAvailable = true
	// Replace back to original name
	replaceRRName(res.Answer, lookupName, origName)
	replaceRRName(res.Ns    , lookupName, origName)
	replaceRRName(res.Extra , lookupName, origName)
	return res, nil
}
