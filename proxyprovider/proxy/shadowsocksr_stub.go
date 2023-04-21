//go:build with_proxyprovider && !with_shadowsocksr

package proxy

import (
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

func (p *ProxyShadowsocksR) GenerateOptions() (*option.Outbound, error) {
	return nil, E.New(`ShadowsocksR is not included in this build, rebuild with -tags with_shadowsocksr`)
}
