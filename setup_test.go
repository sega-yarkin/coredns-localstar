package localstar

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/suite"
)


type SetupTestSuite struct {
	suite.Suite
}

func TestSetupTestSuite(t *testing.T) {
	suite.Run(t, new(SetupTestSuite))
}

func (s *SetupTestSuite) ErrContains(err error, contains string) bool {
	return s.Error(err) && s.Contains(err.Error(), contains)
}

func (s *SetupTestSuite) generateResolvConf(nameserver string) string {
	tmpfile, err := ioutil.TempFile("", "resolv.conf.*")
	if err != nil {
		panic(err)
	}
	content := []byte("nameserver " + nameserver + "\n")
	if _, err := tmpfile.Write(content); err != nil {
		panic(err)
	}
	if err := tmpfile.Close(); err != nil {
		panic(err)
	}
	return tmpfile.Name()
}

func (s *SetupTestSuite) parseConfig(zone, input string) (LocalStar, error) {
	ls := newLocalStar(&dnsserver.Config{Zone: dns.CanonicalName(zone)})
	ls.defaultEndpoints = []string{"10.20.30.40"}
	return ls, parseConfig(caddy.NewTestController("dns", input), &ls)
}

func (s *SetupTestSuite) parseConfigDefaultZone(input string) (LocalStar, error) {
	return s.parseConfig("example.com.", input)
}

func (s *SetupTestSuite) Test_to_zone() {
	var (ls LocalStar; err error)
	
	_, err = s.parseConfigDefaultZone("localstar\n")
	s.ErrContains(err, "parameter is required")

	_, err = s.parseConfigDefaultZone("localstar {\n}\n")
	s.ErrContains(err, "parameter is required")

	ls, err = s.parseConfigDefaultZone(`localstar {
		to_zone corp.net
	}`)
	if s.NoError(err) && s.NotNil(ls) {
		s.Equal(ls.fromZone, "example.com.")
		s.Equal(ls.toZone, "corp.net.")
		s.Equal(ls.toZoneDiff, "")
		s.Equal(ls.endpoints, []string{"10.20.30.40:53"})
	}

	_, err = s.parseConfigDefaultZone(`localstar {
		to_zone
	}`)
	s.ErrContains(err, "Wrong argument count or unexpected line ending after 'to_zone'")

	// test zones diff
	parse := func (from, to string) (LocalStar, error) {
		return s.parseConfig(from, `localstar {
			to_zone ` + to + `
		}`)
	}
	ls, err = parse("sub.corp.net", "corp.net")
	if s.NoError(err) {
		s.Equal(ls.toZoneDiff, "sub.")
	}	
	ls, err = parse("sub1.sub0.corp.net", "corp.net")
	if s.NoError(err) {
		s.Equal(ls.toZoneDiff, "sub1.sub0.")
	}
	ls, err = parse("corp.net", "sub.corp.net")
	s.ErrContains(err, "'to_zone' cannot be equal to or be a child of serving zone")
}

func (s *SetupTestSuite) Test_endpoints() {
	var (ls LocalStar; err error)
	parse0 := func (endpoints []string) (LocalStar, error) {
		return s.parseConfigDefaultZone(`localstar {
			to_zone corp.net
			endpoints ` + strings.Join(endpoints, " ") + `
		}`)
	}

	ls, err = parse0([]string{})
	s.NotNil(ls)
	s.ErrContains(err, "Wrong argument count or unexpected line ending after 'endpoints'")

	parse := func (endpoints []string) []string {
		ls, err := parse0(endpoints)
		if s.Nil(err) && s.NotNil(ls) {
			return ls.endpoints
		}
		return nil
	}
	resolveConf := s.generateResolvConf("10.1.1.1")
	defer os.Remove(resolveConf)

	s.Equal([]string{"1.2.3.4:53"}, parse([]string{"1.2.3.4"}))
	s.Equal([]string{"1.2.3.4:53"}, parse([]string{"1.2.3.4:53"}))
	s.Equal([]string{"1.2.3.4:1053"}, parse([]string{"1.2.3.4:1053"}))
	s.Equal([]string{"1.2.3.4:53"}, parse([]string{"dns://1.2.3.4"}))
	s.Equal([]string{"dns://1.2.3.4:1053"}, parse([]string{"dns://1.2.3.4:1053"}))
	s.Equal([]string{"tls://1.2.3.4:853"}, parse([]string{"tls://1.2.3.4"}))

	s.Equal(
		[]string{"1.2.3.4:53", "tls://1.2.3.4:853", "grpc://1.2.3.4:443", "https://1.2.3.4:443", "10.1.1.1:53"},
		parse([]string{"dns://1.2.3.4", "tls://1.2.3.4", "grpc://1.2.3.4", "https://1.2.3.4", resolveConf}),
	)

	_, err = parse0([]string{"proto://1.2.3.4"})
	s.ErrContains(err, "not an IP address or file")

	// multiple lines
	ls, err = s.parseConfigDefaultZone(`localstar {
		to_zone corp.net
		endpoint tls://1.2.3.4
		endpoints grpc://1.2.3.4
		endpoints dns://1.2.3.4 https://1.2.3.4 ` + resolveConf + `
	}`)
	if s.Nil(err) && s.NotNil(ls) {
		s.Equal(
			[]string{"tls://1.2.3.4:853", "grpc://1.2.3.4:443", "1.2.3.4:53", "https://1.2.3.4:443", "10.1.1.1:53"},
			ls.endpoints,
		)
	}
}

func (s *SetupTestSuite) Test_prefix_len() {
	var (ls LocalStar; err error)
	ls, err = s.parseConfigDefaultZone(`localstar {
		to_zone corp.net
	}`)
	if s.Nil(err) && s.NotNil(ls) {
		s.Equal(1, ls.prefixLen)
	}

	parse := func (l string) (LocalStar, error) {
		return s.parseConfigDefaultZone(`localstar {
			to_zone corp.net
			prefix_len ` + l + `
		}`)
	}

	_, err = parse("")
	s.ErrContains(err, "Wrong argument count or unexpected line ending")
	_, err = parse("string")
	s.ErrContains(err, "invalid number")
	_, err = parse("0")
	s.ErrContains(err, "prefix_len can't be less than 1")
	_, err = parse("-1")
	s.ErrContains(err, "prefix_len can't be less than 1")

	ls, err = parse("5")
	if s.Nil(err) && s.NotNil(ls) {
		s.Equal(5, ls.prefixLen)
	}
}

func (s *SetupTestSuite) Test_timeout() {
	var (ls LocalStar; err error)
	ls, err = s.parseConfigDefaultZone(`localstar {
		to_zone corp.net
	}`)
	if s.Nil(err) && s.NotNil(ls) {
		s.Equal(defaultTimeout, ls.timeout)
	}

	parse := func (t string) (LocalStar, error) {
		return s.parseConfigDefaultZone(`localstar {
			to_zone corp.net
			timeout ` + t + `
		}`)
	}
	parseOk := func (t string) time.Duration {
		ls, err := parse(t)
		if s.Nil(err) && s.NotNil(ls) {
			return ls.timeout
		}
		return 0
	}
	parseErr := func (t string) error {
		_, err = parse(t)
		return err
	}

	s.Equal(1 * time.Millisecond, parseOk("1ms"))
	s.Equal(100 * time.Microsecond, parseOk("100us"))
	s.Equal(1 * time.Hour, parseOk("1h"))
	s.Equal(defaultTimeout, parseOk("0"))

	s.ErrContains(parseErr(""), "Wrong argument count or unexpected line ending")
	s.ErrContains(parseErr("string"), "invalid duration")
	s.ErrContains(parseErr("10"), "invalid duration")
	s.ErrContains(parseErr("-10s"), "timeout can't be negative")
}
