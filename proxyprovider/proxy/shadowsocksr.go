//go:build with_proxyprovider

package proxy

import (
	"net"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

type proxyClashShadowsocksR struct {
	proxyClashDefault `yaml:",inline"`
	//
	Cipher   string `yaml:"cipher,omitempty"`
	Password string `yaml:"password,omitempty"`
	//
	Obfs             string `yaml:"obfs,omitempty"`
	ObfsParam        string `yaml:"obfs-param,omitempty"`
	ObfsParamOld     string `yaml:"obfsparam,omitempty"` // clashR old field support
	Protocol         string `yaml:"protocol,omitempty"`
	ProtocolParam    string `yaml:"protocol-param,omitempty"`
	ProtocolParamOld string `yaml:"protocolparam,omitempty"` // clashR old field support
	//
	UDP *bool `yaml:"udp,omitempty"`
}

type ProxyShadowsocksR struct {
	tag           string
	clashOptions  *proxyClashShadowsocksR
	dialerOptions option.DialerOptions
}

func (p *ProxyShadowsocksR) Tag() string {
	if p.tag == "" {
		p.tag = p.clashOptions.Name
	}
	if p.tag == "" {
		p.tag = net.JoinHostPort(p.clashOptions.Server, p.clashOptions.ServerPort.Value)
	}
	return p.tag
}

func (p *ProxyShadowsocksR) Type() string {
	return C.TypeShadowsocksR
}

func (p *ProxyShadowsocksR) SetClashOptions(options any) bool {
	clashOptions, ok := options.(proxyClashShadowsocksR)
	if !ok {
		return false
	}
	p.clashOptions = &clashOptions
	return true
}

func (p *ProxyShadowsocksR) GetClashType() string {
	return p.clashOptions.Type
}

func (p *ProxyShadowsocksR) SetDialerOptions(dialer option.DialerOptions) {
	p.dialerOptions = dialer
}
