//go:build with_proxyprovider

package proxy

import (
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
	E "github.com/sagernet/sing/common/exceptions"
	N "github.com/sagernet/sing/common/network"
	"net"
	"strconv"
)

type proxyClashShadowsocks struct {
	proxyClashDefault `yaml:",inline"`
	//
	Cipher   string `yaml:"cipher,omitempty"`
	Password string `yaml:"password,omitempty"`
	//
	UDP        bool `yaml:"udp,omitempty"`
	UDPOverTCP bool `yaml:"udp-over-tcp,omitempty"`
}

type ProxyShadowsocks struct {
	tag           string
	clashOptions  *proxyClashShadowsocks
	dialerOptions option.DialerOptions
}

func (p *ProxyShadowsocks) Tag() string {
	if p.tag == "" {
		p.tag = p.clashOptions.Name
	}
	if p.tag == "" {
		p.tag = net.JoinHostPort(p.clashOptions.Server, strconv.Itoa(int(p.clashOptions.ServerPort)))
	}
	return p.tag
}

func (p *ProxyShadowsocks) Type() string {
	return C.TypeShadowsocks
}

func (p *ProxyShadowsocks) SetClashOptions(options any) bool {
	clashOptions, ok := options.(proxyClashShadowsocks)
	if !ok {
		return false
	}
	p.clashOptions = &clashOptions
	return true
}

func (p *ProxyShadowsocks) GetClashType() string {
	return p.clashOptions.Type
}

func (p *ProxyShadowsocks) SetDialerOptions(dialer option.DialerOptions) {
	p.dialerOptions = dialer
}

func (p *ProxyShadowsocks) GenerateOptions() (*option.Outbound, error) {
	if !p.checkMethod() {
		return nil, E.New("shadowsocks cipher: ", p.clashOptions.Cipher, " is not supported in sing-box")
	}

	opt := &option.Outbound{
		Tag:  p.Tag(),
		Type: C.TypeShadowsocks,
		ShadowsocksOptions: option.ShadowsocksOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     p.clashOptions.Server,
				ServerPort: p.clashOptions.ServerPort,
			},
			Method:   p.clashOptions.Cipher,
			Password: p.clashOptions.Password,
			//
			DialerOptions: p.dialerOptions,
		},
	}

	if !p.clashOptions.UDP {
		opt.ShadowsocksOptions.Network = N.NetworkTCP
	}

	if p.clashOptions.UDPOverTCP {
		opt.ShadowsocksOptions.UDPOverTCPOptions = &option.UDPOverTCPOptions{
			Enabled: true,
			Version: 1,
		}
	}

	switch p.clashOptions.IPVersion {
	case "dual":
	case "ipv4":
		opt.ShadowsocksOptions.DialerOptions.DomainStrategy = option.DomainStrategy(dns.DomainStrategyUseIPv4)
	case "ipv6":
		opt.ShadowsocksOptions.DialerOptions.DomainStrategy = option.DomainStrategy(dns.DomainStrategyUseIPv6)
	case "ipv4-prefer":
		opt.ShadowsocksOptions.DialerOptions.DomainStrategy = option.DomainStrategy(dns.DomainStrategyPreferIPv4)
	case "ipv6-prefer":
		opt.ShadowsocksOptions.DialerOptions.DomainStrategy = option.DomainStrategy(dns.DomainStrategyPreferIPv6)
	default:
	}

	return opt, nil
}

//

func (p *ProxyShadowsocks) checkMethod() bool {
	switch p.clashOptions.Cipher {
	case "aes-128-gcm":
	case "aes-192-gcm":
	case "aes-256-gcm":
	case "aes-128-cfb":
	case "aes-192-cfb":
	case "aes-256-cfb":
	case "aes-128-ctr":
	case "aes-192-ctr":
	case "aes-256-ctr":
	case "rc4-md5":
	case "chacha20-ietf":
	case "xchacha20":
	case "chacha20-ietf-poly1305":
	case "xchacha20-ietf-poly1305":
	case "2022-blake3-aes-128-gcm":
	case "2022-blake3-aes-256-gcm":
	case "2022-blake3-chacha20-poly1305":
	default:
		return false
	}
	return true
}
