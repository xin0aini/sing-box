package outbound

import (
	"context"
	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"math/rand"
	"net"
	"time"
)

var (
	_ adapter.Outbound      = (*Random)(nil)
	_ adapter.OutboundGroup = (*Random)(nil)
)

type Random struct {
	myOutboundAdapter
	tags        []string
	outbounds   map[string]adapter.Outbound
	outboundNum uint
}

func NewRandom(router adapter.Router, logger log.ContextLogger, tag string, options option.RandomOutboundOptions) (*Random, error) {
	outbound := &Random{
		myOutboundAdapter: myOutboundAdapter{
			protocol: C.TypeRandom,
			router:   router,
			logger:   logger,
			tag:      tag,
		},
		tags:      options.Outbounds,
		outbounds: make(map[string]adapter.Outbound),
	}
	if len(outbound.outbounds) == 0 {
		return nil, E.New("missing outbounds")
	}
	return outbound, nil
}

func (r *Random) Network() []string {
	return []string{N.NetworkTCP, N.NetworkUDP}
}

func (r *Random) Start() error {
	outboundNum := uint(0)
	for i, tag := range r.tags {
		detour, loaded := r.router.Outbound(tag)
		if !loaded {
			return E.New("outbound ", i, " not found: ", tag)
		}
		r.outbounds[tag] = detour
		outboundNum++
	}
	r.outboundNum = outboundNum

	return nil
}

func (r *Random) GetRandomIndex() uint {
	return uint(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(int(r.outboundNum)))
}

func (r *Random) GetRandomOutbound() adapter.Outbound {
	outboundTag := r.tags[r.GetRandomIndex()]
	return r.outbounds[outboundTag]
}

func (r *Random) All() []string {
	return r.tags
}

func (r *Random) Now() string {
	return r.tags[r.GetRandomIndex()]
}

func (r *Random) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return r.GetRandomOutbound().DialContext(ctx, network, destination)
}

func (r *Random) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return r.GetRandomOutbound().ListenPacket(ctx, destination)
}

func (r *Random) NewConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext) error {
	return r.GetRandomOutbound().NewConnection(ctx, conn, metadata)
}

func (r *Random) NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext) error {
	return r.GetRandomOutbound().NewPacketConnection(ctx, conn, metadata)
}
