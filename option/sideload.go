package option

type SideLoadOutboundOptions struct {
	DialerOptions
	ServerOptions
	ListenPort      uint16           `json:"listen_port"`
	ListenNetwork   NetworkList      `json:"listen_network"`
	Command         Listable[string] `json:"command,omitempty"`
	Env             Listable[string] `json:"env"`
	Socks5ProxyPort uint16           `json:"socks5_proxy_port"`
	Network         NetworkList      `json:"network"`
}
