//go:build with_proxyprovider

package proxy

import (
	"net"
	"strconv"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
	E "github.com/sagernet/sing/common/exceptions"
)

type proxyClashHTTP struct {
	proxyClashDefault `yaml:",inline"`
	//
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	//
	TLS            bool   `yaml:"tls,omitempty"`
	SkipCertVerify bool   `yaml:"skip-cert-verify,omitempty"`
	ServerName     string `yaml:"sni,omitempty"`
	FingerPrint    string `yaml:"fingerprint,omitempty"`
}

type ProxyHTTP struct {
	tag           string
	clashOptions  *proxyClashHTTP
	dialerOptions option.DialerOptions
}

func (p *ProxyHTTP) Tag() string {
	if p.tag == "" {
		p.tag = p.clashOptions.Name
	}
	if p.tag == "" {
		p.tag = net.JoinHostPort(p.clashOptions.Server, p.clashOptions.ServerPort.Value)
	}
	return p.tag
}

func (p *ProxyHTTP) Type() string {
	return C.TypeHTTP
}

func (p *ProxyHTTP) SetClashOptions(options any) bool {
	clashOptions, ok := options.(proxyClashHTTP)
	if !ok {
		return false
	}
	p.clashOptions = &clashOptions
	return true
}

func (p *ProxyHTTP) GetClashType() string {
	return p.clashOptions.Type
}

func (p *ProxyHTTP) SetDialerOptions(dialer option.DialerOptions) {
	p.dialerOptions = dialer
}

func (p *ProxyHTTP) GenerateOptions() (*option.Outbound, error) {
	serverPort, err := strconv.ParseUint(p.clashOptions.ServerPort.Value, 10, 16)
	if err != nil {
		return nil, E.Cause(err, "fail to parse port")
	}

	opt := &option.Outbound{
		Tag:  p.Tag(),
		Type: C.TypeHTTP,
		HTTPOptions: option.HTTPOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     p.clashOptions.Server,
				ServerPort: uint16(serverPort),
			},
			Username: p.clashOptions.Username,
			Password: p.clashOptions.Password,
			//
			DialerOptions: p.dialerOptions,
		},
	}

	if p.clashOptions.TLS {
		opt.HTTPOptions.TLS = &option.OutboundTLSOptions{
			Enabled: true,
		}
		if p.clashOptions.ServerName != "" {
			opt.HTTPOptions.TLS.ServerName = p.clashOptions.ServerName
		}
		if p.clashOptions.SkipCertVerify {
			opt.HTTPOptions.TLS.Insecure = true
		}
		if p.clashOptions.FingerPrint != "" {
			if !GetTag("with_utls") {
				return nil, E.New(`uTLS is not included in this build, rebuild with -tags with_utls`)
			}
			opt.HTTPOptions.TLS.UTLS = &option.OutboundUTLSOptions{
				Fingerprint: p.clashOptions.FingerPrint,
			}
		}
	}

	switch p.clashOptions.IPVersion {
	case "dual":
	case "ipv4":
		opt.HTTPOptions.DialerOptions.DomainStrategy = option.DomainStrategy(dns.DomainStrategyUseIPv4)
	case "ipv6":
		opt.HTTPOptions.DialerOptions.DomainStrategy = option.DomainStrategy(dns.DomainStrategyUseIPv6)
	case "ipv4-prefer":
		opt.HTTPOptions.DialerOptions.DomainStrategy = option.DomainStrategy(dns.DomainStrategyPreferIPv4)
	case "ipv6-prefer":
		opt.HTTPOptions.DialerOptions.DomainStrategy = option.DomainStrategy(dns.DomainStrategyPreferIPv6)
	default:
	}

	return opt, nil
}
