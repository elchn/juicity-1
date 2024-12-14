package http

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/daeuniverse/softwind/netproxy"
	tls2 "github.com/daeuniverse/softwind/transport/tls"
)

// HttpProxy is an HTTP/HTTPS proxy.
type HttpProxy struct {
	https     bool
	transport bool
	Addr      string
	Host      string
	Path      string
	HaveAuth  bool
	Username  string
	Password  string
	dialer    netproxy.Dialer
}

func NewHTTPProxy(u *url.URL, forward netproxy.Dialer) (netproxy.Dialer, error) {
	s := new(HttpProxy)
	s.Addr = u.Host
	s.Path = u.Path
	if !strings.HasPrefix(s.Path, "/") {
		s.Path = "/" + s.Path
	}
	s.Host = u.Query().Get("host")
	s.dialer = forward
	if u.User != nil {
		s.HaveAuth = true
		s.Username = u.User.Username()
		s.Password, _ = u.User.Password()
	}
	s.transport, _ = strconv.ParseBool(u.Query().Get("transport"))
	if u.Scheme == "https" {
		s.https = true
		serverName := u.Query().Get("sni")
		if serverName == "" {
			serverName = u.Hostname()
		}

		tlsImplementation := "tls"
		if u.Query().Get("tlsImplementation") != "" {
			tlsImplementation = u.Query().Get("tlsImplementation")
		}
		alpn := []string{"h2,http/1.1"}
		if u.Query().Get("alpn") != "" {
			alpn = []string{u.Query().Get("alpn")}
		}
		u := url.URL{
			Scheme: tlsImplementation,
			Host:   s.Addr,
			RawQuery: url.Values{
				"sni":           []string{serverName},
				"allowInsecure": []string{u.Query().Get("allowInsecure")},
				"utlsImitate":   []string{u.Query().Get("utlsImitate")},
				"alpn":          alpn,
			}.Encode(),
		}
		var err error
		s.dialer, err = tls2.NewTls(u.String(), s.dialer)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *HttpProxy) Dial(network, addr string) (netproxy.Conn, error) {
	magicNetwork, err := netproxy.ParseMagicNetwork(network)
	if err != nil {
		return nil, err
	}
	switch magicNetwork.Network {
	case "tcp":
		return NewConn(s.dialer, s, addr, network), nil
	case "udp":
		return nil, netproxy.UnsupportedTunnelTypeError
	default:
		return nil, fmt.Errorf("%w: %v", netproxy.UnsupportedTunnelTypeError, network)
	}
}
