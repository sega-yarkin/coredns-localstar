package localstar

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/suite"
)

type SimpleDNSProviderTestSuite struct {
	suite.Suite
}

func TestSimpleDNSProviderTestSuite(t *testing.T) {
	suite.Run(t, new(SimpleDNSProviderTestSuite))
}

func (s *SimpleDNSProviderTestSuite) Test_exchange() {
	// test it works with Google Public DNS
	p := new(simpleDNSProvider)
	p.Init([]string{"8.8.8.8:53"}, 4 * time.Second)
	msg := new(dns.Msg)
	msg.SetQuestion("publicdns.google.com.", dns.TypeA)
	res, err := p.Exchange(context.Background(), msg)
	if s.NoError(err) && s.NotNil(res) {
		s.Equal(dns.RcodeSuccess, res.Rcode)
		var ips []string
		for _, rr := range res.Answer {
			if rrA, ok := rr.(*dns.A); ok {
				ips = append(ips, rrA.A.String())
			}
		}
		expectedIps := []string{"8.8.8.8", "8.8.4.4"}
		s.ElementsMatch(expectedIps, ips)
	}
}

func (s *SimpleDNSProviderTestSuite) Test_exchange_timeout() {
	var (err error; dur time.Duration; ctx context.Context; cancel context.CancelFunc)

	exch := func (t time.Duration, ctx context.Context) (time.Duration, error) {
		p := new(simpleDNSProvider)
		p.Init([]string{"127.126.125.124:53"}, t)
		msg := new(dns.Msg)
		msg.SetQuestion("localhost.", dns.TypeA)
		start := time.Now()
		res, err := p.Exchange(ctx, msg)
		dur := time.Since(start)
		s.Nil(res)
		return dur, err
	}

	dur, err = exch(30*time.Millisecond, context.Background())
	if s.Error(err) {
		s.InDelta(30*time.Millisecond, dur, float64(5*time.Millisecond))
		var ne *net.OpError
		if s.ErrorAs(err, &ne) {
			s.True(ne.Timeout())
		}
	}

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Millisecond)
	dur, err = exch(1*time.Second, ctx)
	if s.Error(err) {
		s.InDelta(30*time.Millisecond, dur, float64(5*time.Millisecond))
		// NOTE: Context is not supported by underlying API, so we get back net error
		var ne *net.OpError
		if s.ErrorAs(err, &ne) {
			s.True(ne.Timeout())
		}
	}
	cancel()
}
