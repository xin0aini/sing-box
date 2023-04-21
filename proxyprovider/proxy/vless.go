//go:build with_proxyprovider

package proxy

import (
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"net"
	"strconv"
)

type proxyClashVLESS struct {
	proxyClashDefault `yaml:",inline"`
	//
	UUID                string `yaml:"uuid"`
	AlterID             int    `yaml:"alterId"`
	Cipher              string `yaml:"cipher"`
	UDP                 bool   `yaml:"udp,omitempty"`
	TLS                 bool   `yaml:"tls,omitempty"`
	SkipCertVerify      bool   `yaml:"skip-cert-verify,omitempty"`
	Fingerprint         string `yaml:"fingerprint,omitempty"`
	ClientFingerprint   string `yaml:"client-fingerprint,omitempty"`
	ServerName          string `yaml:"servername,omitempty"`
	PacketEncoding      string `yaml:"packet-encoding,omitempty"`
	GlobalPadding       bool   `yaml:"global-padding,omitempty"`
	AuthenticatedLength bool   `yaml:"authenticated-length,omitempty"`
	//
	Network string `yaml:"network,omitempty"`
	//
	WSOptions    *proxyClashVMessWSOptions    `yaml:"ws-opts,omitempty"`
	HTTPOptions  *proxyClashVMessHTTPOptions  `yaml:"http-opts,omitempty"`
	HTTP2Options *proxyClashVMessHTTP2Options `yaml:"h2-opts,omitempty"`
	GrpcOptions  *proxyClashVMessGrpcOptions  `yaml:"grpc-opts,omitempty"`
	//
	RealityOptions *proxyClashVMessRealityOptions `proxy:"reality-opts,omitempty"`
}

type ProxyVLESS struct {
	tag           string
	clashOptions  *proxyClashVLESS
	dialerOptions option.DialerOptions
}

func (p *ProxyVLESS) Tag() string {
	if p.tag == "" {
		p.tag = p.clashOptions.Name
	}
	if p.tag == "" {
		p.tag = net.JoinHostPort(p.clashOptions.Server, strconv.Itoa(int(p.clashOptions.ServerPort)))
	}
	return p.tag
}

func (p *ProxyVLESS) Type() string {
	return C.TypeVLESS
}

func (p *ProxyVLESS) SetClashOptions(options any) bool {
	clashOptions, ok := options.(proxyClashVLESS)
	if !ok {
		return false
	}
	p.clashOptions = &clashOptions
	return true
}

func (p *ProxyVLESS) GetClashType() string {
	return p.clashOptions.Type
}

func (p *ProxyVLESS) SetDialerOptions(dialer option.DialerOptions) {
	p.dialerOptions = dialer
}

func (p *ProxyVLESS) GenerateOptions() (*option.Outbound, error) {
	return nil, nil
}
