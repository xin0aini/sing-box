//go:build with_proxyprovider

package proxy

import (
	E "github.com/sagernet/sing/common/exceptions"
)

type proxyClashDefault struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Server     string `yaml:"server"`
	ServerPort uint16 `yaml:"port"`
	//
	IPVersion string `yaml:"ip-version,omitempty"`
}

type ProxyClashOptions struct {
	Type                string                 `yaml:"type"`
	HTTPOptions         proxyClashHTTP         `yaml:"-"`
	SocksOptions        proxyClashSocks        `yaml:"-"`
	ShadowsocksOptions  proxyClashShadowsocks  `yaml:"-"`
	ShadowsocksROptions proxyClashShadowsocksR `yaml:"-"`
	VMessOptions        proxyClashVMess        `yaml:"-"`
	TrojanOptions       proxyClashTrojan       `yaml:"-"`
}

type _proxyClashOptions ProxyClashOptions

func (p *ProxyClashOptions) UnmarshalYAML(unmarshal func(any) error) error {
	var raw _proxyClashOptions
	if err := unmarshal(&raw); err != nil {
		return err
	}
	*p = ProxyClashOptions{
		Type: raw.Type,
	}
	switch raw.Type {
	case "http":
		return unmarshal(&p.HTTPOptions)
	case "socks5":
		return unmarshal(&p.SocksOptions)
	case "ss":
		return unmarshal(&p.ShadowsocksOptions)
	case "ssr":
		return unmarshal(&p.ShadowsocksROptions)
	case "vmess":
		return unmarshal(&p.VMessOptions)
	case "trojan":
		return unmarshal(&p.TrojanOptions)
	default:
		// return E.New("unsupported clash proxy type: ", raw.Type)
		return nil
	}
}

func (p *ProxyClashOptions) MarshalYAML() (any, error) {
	switch p.Type {
	case "http":
		return p.HTTPOptions, nil
	case "socks5":
		return p.SocksOptions, nil
	case "ss":
		return p.ShadowsocksOptions, nil
	case "ssr":
		return p.ShadowsocksROptions, nil
	case "vmess":
		return p.VMessOptions, nil
	case "trojan":
		return p.TrojanOptions, nil
	default:
		return nil, E.New("unsupported clash proxy type: ", p.Type)
	}
}

func (p *ProxyClashOptions) ToProxy() (Proxy, error) {
	var opt Proxy
	switch p.Type {
	case "http":
		opt = &ProxyHTTP{}
		opt.SetClashOptions(p.HTTPOptions)
	case "socks5":
		opt = &ProxySocks{}
		opt.SetClashOptions(p.SocksOptions)
	case "ss":
		opt = &ProxyShadowsocks{}
		opt.SetClashOptions(p.ShadowsocksOptions)
	case "ssr":
		opt = &ProxyShadowsocksR{}
		opt.SetClashOptions(p.ShadowsocksROptions)
	case "vmess":
		opt = &ProxyVMess{}
		opt.SetClashOptions(p.VMessOptions)
	case "trojan":
		opt = &ProxyTrojan{}
		opt.SetClashOptions(p.VMessOptions)
	default:
		return nil, E.New("unsupported clash proxy type: ", p.Type)
	}
	opt.Tag()
	return opt, nil
}

type ClashConfig struct {
	Proxies []ProxyClashOptions `yaml:"proxies"`
}
