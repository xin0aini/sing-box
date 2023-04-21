//go:build with_proxyprovider

package proxy

import "github.com/sagernet/sing-box/option"

type Proxy interface {
	Tag() string
	Type() string
	SetClashOptions(options any) bool
	GetClashType() string
	SetDialerOptions(dialer option.DialerOptions)
	GenerateOptions() (*option.Outbound, error)
}
