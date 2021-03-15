package localstar

import (
	"context"
	"time"

	"github.com/miekg/dns"
)

type dnsProvider interface {
	Init(endpoints []string, timeout time.Duration) error
	Exchange(ctx context.Context, req *dns.Msg) (resp *dns.Msg, err error)
}

type dnsProviderInit func ([]string, time.Duration) error
type dnsProviderExchange func (context.Context, *dns.Msg) (*dns.Msg, error)

type simpleDNSProvider struct {
	endpoint string
	timeout  time.Duration
}

// stubDNSProvider used in tests
type stubDNSProvider struct {
	initCb     dnsProviderInit
	exchangeCb dnsProviderExchange
}

// TODO: Add forward pluging provider


func (p *simpleDNSProvider) Init(endpoints []string, timeout time.Duration) error {
	p.endpoint = endpoints[0]
	p.timeout = timeout
	return nil
}

func (p *simpleDNSProvider) Exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	client := &dns.Client{Net: "udp", Timeout: p.timeout}
	res, _, err := client.ExchangeContext(ctx, msg, p.endpoint)
	return res, err
}


func (p *stubDNSProvider) Init(endpoints []string, timeout time.Duration) error {
	if p.initCb != nil {
		return p.initCb(endpoints, timeout)
	}
	return nil
}

func (p *stubDNSProvider) Exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	if p.exchangeCb != nil {
		return p.exchangeCb(ctx, msg)
	}
	return msg, nil
}
