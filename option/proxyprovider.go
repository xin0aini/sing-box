package option

import (
	"github.com/sagernet/sing-box/common/json"
	C "github.com/sagernet/sing-box/constant"
	E "github.com/sagernet/sing/common/exceptions"
)

type ProxyProviderOptions struct {
	Tag                  string                                    `json:"tag"`
	URL                  string                                    `json:"url"`
	CacheFile            string                                    `json:"cache_file"`
	ForceUpdate          Duration                                  `json:"force_update"`
	DNS                  string                                    `json:"dns"`
	Filter               *ProxyProviderFilterOptions               `json:"filter"`
	DefaultOutbound      string                                    `json:"default_outbound"`
	RequestDialerOptions *ProxyProviderRequestDialerOptions        `json:"request_dialer"`
	DialerOptions        *DialerOptions                            `json:"dialer"`
	CustomGroup          Listable[ProxyProviderCustomGroupOptions] `json:"custom_group"`
}

type ProxyProviderCustomGroupOptions struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
	ProxyProviderFilterOptions
	SelectorOptions SelectorOutboundOptions `json:"-"`
	URLTestOptions  URLTestOutboundOptions  `json:"-"`
}

type _proxyProviderCustomGroupOptions ProxyProviderCustomGroupOptions

func (p *ProxyProviderCustomGroupOptions) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, (*_proxyProviderCustomGroupOptions)(p))
	if err != nil {
		return err
	}
	var h any
	switch p.Type {
	case C.TypeSelector:
		h = &p.SelectorOptions
	case C.TypeURLTest:
		h = &p.URLTestOptions
	default:
		return E.New("unknown group type: ", p.Type)
	}
	err = UnmarshallExcluded(data, (*_proxyProviderCustomGroupOptions)(p), h)
	if err != nil {
		return E.Cause(err, "proxyprovider options")
	}
	return nil
}

func (p ProxyProviderCustomGroupOptions) MarshalJSON() ([]byte, error) {
	var v any
	switch p.Type {
	case C.TypeSelector:
		v = p.SelectorOptions
	case C.TypeURLTest:
		v = p.URLTestOptions
	default:
		return nil, E.New("unknown group type: ", p.Type)
	}
	return MarshallObjects((*_proxyProviderCustomGroupOptions)(&p), v)
}

type ProxyProviderFilterOptions struct {
	Rule      Listable[*Filter] `json:"rule"`
	WhiteMode bool              `json:"white_mode"`
}

type ProxyProviderRequestDialerOptions struct {
	BindInterface      string         `json:"bind_interface,omitempty"`
	Inet4BindAddress   *ListenAddress `json:"inet4_bind_address,omitempty"`
	Inet6BindAddress   *ListenAddress `json:"inet6_bind_address,omitempty"`
	ProtectPath        string         `json:"protect_path,omitempty"`
	RoutingMark        int            `json:"routing_mark,omitempty"`
	ReuseAddr          bool           `json:"reuse_addr,omitempty"`
	ConnectTimeout     Duration       `json:"connect_timeout,omitempty"`
	TCPFastOpen        bool           `json:"tcp_fast_open,omitempty"`
	UDPFragment        *bool          `json:"udp_fragment,omitempty"`
	UDPFragmentDefault bool           `json:"-"`
}
