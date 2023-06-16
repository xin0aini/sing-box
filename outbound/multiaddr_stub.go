//go:build !with_multiaddr

package outbound

import (
	"context"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

func NewMultiAddr(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.MultiAddrOutboundOptions) (adapter.Outbound, error) {
	return nil, E.New(`MultiAddr is not included in this build, rebuild with -tags with_multiaddr`)
}
