package option

type MultiAddrOutboundOptions struct {
	Addresses Listable[MultiAddrOptions] `json:"addresses"`
	Network   NetworkList                `json:"network,omitempty"`
	DialerOptions
}

type MultiAddrOptions struct {
	IP        string `json:"ip,omitempty"`
	IPRange   string `json:"ip_range,omitempty"`
	CIDR      string `json:"cidr,omitempty"`
	Port      uint16 `json:"port,omitempty"`
	PortRange string `json:"port_range,omitempty"`
}
