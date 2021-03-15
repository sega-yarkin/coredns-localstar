package localstar

import (
	"context"
	"errors"
	"testing"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/suite"
)

type LocalStarTestSuite struct {
	suite.Suite
}

func TestLocalStarTestSuite(t *testing.T) {
	suite.Run(t, new(LocalStarTestSuite))
}

func (s *LocalStarTestSuite) Test_newLocalStar() {
	zoneName := "zone.name"
	config := &dnsserver.Config{Zone: dns.CanonicalName(zoneName)}
	ls := newLocalStar(config)
	s.Equal("zone.name.", ls.fromZone)
}

func (s *LocalStarTestSuite) Test_calcZoneDiff() {
	for _, t := range []struct {
		from, to, diff string
	}{
		{"c.b.a", "b.a", "c."},
		{"d.c.b.a", "b.a", "d.c."},
		{"d.c.b.a", "a", "d.c.b."},
		{"d.c.b.aa", "a", ""},
		{"d.c.b.a", "d.c.b", ""},
		{"dev.corp.net", "corp.net", "dev."},
		{"r0.dev.corp.net", "corp.net", "r0.dev."},
	}{
		if diff := calcZoneDiff(t.from, t.to); diff != t.diff {
			s.Fail("Bad zones diff", "Bad zones diff for '%s'->'%s', expect '%s', got '%s'", t.from, t.to, t.diff, diff)
		}
	}
}

func (s *LocalStarTestSuite) Test_getLookupName() {
	var tests = []struct {
		from string
		to string
		prefix int
		qname string
		expect string
		err error
	}{
		{"dev.corp.net", "corp.net", 1, "host.dev.corp.net", "host.corp.net", nil},
		{"dev.corp.net", "corp.net", 1, "host.dev.dev.corp.net", "", errLoopRequest},
		{"dev.corp.net", "net", 1, "host.dev.dev.corp.net", "dev.net", nil},
		{"dev.corp.net", "net", 2, "host.dev.dev.corp.net", "host.dev.net", nil},
		{"dev.corp.net", "my.com", 1, "host.dev.corp.net", "host.my.com", nil},
		{"dev.corp.net", "my.com", 1, "d.c.b.a.dev.corp.net", "a.my.com", nil},
		{"dev.corp.net", "my.com", 2, "d.c.b.a.dev.corp.net", "b.a.my.com", nil},
		{"dev.corp.net", "my.com", 3, "d.c.b.a.dev.corp.net", "c.b.a.my.com", nil},
		{"dev.corp.net", "my.com", 4, "d.c.b.a.dev.corp.net", "d.c.b.a.my.com", nil},
	}
	for _, t := range tests {
		t.from = dns.Fqdn(t.from)
		t.to = dns.Fqdn(t.to)
		t.qname = dns.Fqdn(t.qname)
		t.expect = dns.Fqdn(t.expect)
		ls := LocalStar{fromZone: t.from, toZone: t.to, prefixLen: t.prefix, toZoneDiff: calcZoneDiff(t.from, t.to)}
		lname, err := ls.getLookupName(t.qname)
		if t.err != nil {
			s.ErrorIs(err, t.err, t)
		} else {
			_ = s.NoError(err, t) && s.Equal(t.expect, lname, t)
		}
	}
}

func (s *LocalStarTestSuite) Test_lookupOnExternalDNS() {
	var (err error; res *dns.Msg)
	ctx0 := context.Background()
	err0 := errors.New("err0")

	doLookup := func (origName, lookupName string, cb dnsProviderExchange) (*dns.Msg, error) {
		ls := LocalStar{provider: &stubDNSProvider{exchangeCb: cb}}
		msg := new(dns.Msg)
		msg.SetQuestion(origName, dns.TypeA)
		msg.Id = 1
		return ls.lookupOnExternalDNS(ctx0, lookupName, origName, msg)
	}
	msgQname := func (msg *dns.Msg) string {
		return (&request.Request{Req: msg}).Name()
	}

	// right msg sent
	res, err = doLookup("test.example.com.", "test.corp.net.",
		func (ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
			s.Equal("test.corp.net.", msgQname(msg))
			s.Equal(ctx0, ctx)
			return nil, err0
		},
	)
	s.Nil(res)
	s.Equal(err0, err)

	// response updated properly
	res, err = doLookup("test.example.com.", "test.corp.net.",
		func (ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
			res := &dns.Msg{
				MsgHdr: dns.MsgHdr{Id: 1000, Authoritative: true},
				Answer: []dns.RR{
					&dns.A{Hdr: dns.RR_Header{Name: "test.corp.net."}},
					&dns.A{Hdr: dns.RR_Header{Name: "a.test.corp.net."}},
					&dns.CNAME{Hdr: dns.RR_Header{Name: "test.corp.net."}},
				},
				Ns: []dns.RR{
					&dns.SOA{Hdr: dns.RR_Header{Name: "test.corp.net."}},
				},
				Extra: []dns.RR{
					&dns.A{Hdr: dns.RR_Header{Name: "test.corp.net."}},
					&dns.A{Hdr: dns.RR_Header{Name: "a.test.corp.net."}},
					&dns.CNAME{Hdr: dns.RR_Header{Name: "test.corp.net."}},
				},
			}
			return res, nil
		},
	)
	if s.Nil(err) && s.NotNil(res) {
		s.Equal(uint16(1), res.Id)
		s.Equal(false, res.Authoritative)
		s.Equal("test.example.com.", msgQname(res))
		if s.Equal(3, len(res.Answer)) {
			s.Equal("test.example.com.", res.Answer[0].Header().Name)
			s.Equal("a.test.corp.net.", res.Answer[1].Header().Name)
			s.Equal("test.example.com.", res.Answer[2].Header().Name)
		}
		if s.Equal(1, len(res.Ns)) {
			s.Equal("test.example.com.", res.Ns[0].Header().Name)
		}
		if s.Equal(3, len(res.Extra)) {
			s.Equal("test.example.com.", res.Extra[0].Header().Name)
			s.Equal("a.test.corp.net.", res.Extra[1].Header().Name)
			s.Equal("test.example.com.", res.Extra[2].Header().Name)
		}
	}
}
