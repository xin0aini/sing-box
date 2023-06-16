//go:build with_multiaddr

package outbound

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/dialer"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

var _ adapter.Outbound = (*MultiAddr)(nil)

type MultiAddr struct {
	myOutboundAdapter
	ctx        context.Context
	dialer     N.Dialer
	multiAddrs []*multiAddr
}

func NewMultiAddr(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.MultiAddrOutboundOptions) (*MultiAddr, error) {
	m := &MultiAddr{
		myOutboundAdapter: myOutboundAdapter{
			protocol:     C.TypeMultiAddr,
			network:      options.Network.Build(),
			router:       router,
			logger:       logger,
			tag:          tag,
			dependencies: withDialerDependency(options.DialerOptions),
		},
		ctx:    ctx,
		dialer: dialer.New(router, options.DialerOptions),
	}

	if options.Addresses == nil || len(options.Addresses) == 0 {
		return nil, E.New("no address found")
	}
	mas := make([]*multiAddr, 0)
	for _, addr := range options.Addresses {
		ma, err := newMultiAddr(addr)
		if err != nil {
			return nil, err
		}
		mas = append(mas, ma)
	}
	m.multiAddrs = mas

	return m, nil
}

func (m *MultiAddr) Tag() string {
	return m.tag
}

func (m *MultiAddr) Type() string {
	return C.TypeMultiAddr
}

func (m *MultiAddr) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	ctx, metadata := adapter.AppendContext(ctx)
	metadata.Outbound = m.tag
	metadata.Destination = destination
	destination = m.getDestination(destination)
	network = N.NetworkName(network)
	switch network {
	case N.NetworkTCP:
		m.logger.InfoContext(ctx, "outbound connection to ", destination)
	case N.NetworkUDP:
		m.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	}
	return m.dialer.DialContext(ctx, network, destination)
}

func (m *MultiAddr) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	ctx, metadata := adapter.AppendContext(ctx)
	metadata.Outbound = m.tag
	metadata.Destination = destination
	destination = m.getDestination(destination)
	m.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	conn, err := m.dialer.ListenPacket(ctx, destination)
	if err != nil {
		return nil, err
	}
	return &overridePacketConn{bufio.NewPacketConn(conn), destination}, nil
}

func (m *MultiAddr) NewConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext) error {
	return NewConnection(ctx, m, conn, metadata)
}

func (m *MultiAddr) NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext) error {
	return NewPacketConnection(ctx, m, conn, metadata)
}

func (m *MultiAddr) getDestination(destination M.Socksaddr) M.Socksaddr {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return m.multiAddrs[r.Intn(len(m.multiAddrs))].getAddr(destination)
}

type multiAddr struct {
	ip        *netip.Addr
	prefix    *netip.Prefix
	port      uint16
	startPort uint16
	endPort   uint16
}

func newMultiAddr(options option.MultiAddrOptions) (*multiAddr, error) {
	m := &multiAddr{}
	done := false
	if options.PortRange != "" {
		sub := strings.SplitN(options.PortRange, ":", 2)
		if sub[0] == "" && sub[1] == "" {
			return nil, E.New("invalid port range: ", options.PortRange)
		}
		var (
			startPort uint16
			endPort   uint16
		)
		if sub[0] == "" {
			startPort = 1
		} else {
			startPortUint64, err := strconv.ParseUint(sub[0], 10, 16)
			if err != nil {
				return nil, E.Cause(err, "invalid port range: ", options.PortRange)
			}
			startPort = uint16(startPortUint64)
		}
		if sub[1] == "" {
			endPort = 65535
		} else {
			endPortUint64, err := strconv.ParseUint(sub[1], 10, 16)
			if err != nil {
				return nil, E.Cause(err, "invalid port range: ", options.PortRange)
			}
			endPort = uint16(endPortUint64)
		}
		if startPort > endPort {
			return nil, E.New("invalid port range: ", options.PortRange)
		}
		m.startPort = startPort
		m.endPort = endPort
		done = true
	}
	if options.Port > 0 {
		if options.Port > 65535 {
			return nil, E.New("invalid port: ", options.Port)
		}
		m.port = options.Port
		done = true
	}
	if options.CIDR != "" {
		prefix, err := netip.ParsePrefix(options.CIDR)
		if err != nil {
			return nil, E.Cause(err, "invalid cidr: ", options.CIDR)
		}
		m.prefix = new(netip.Prefix)
		*m.prefix = prefix.Masked()
		done = true
	}
	if options.IP != "" {
		ip, err := netip.ParseAddr(options.IP)
		if err != nil {
			return nil, E.Cause(err, "invalid ip: ", options.IP)
		}
		m.ip = new(netip.Addr)
		*m.ip = ip
		done = true
	}
	if !done {
		return nil, E.New("invalid address: ", fmt.Sprint(options))
	}
	return m, nil
}

func (m *multiAddr) getAddr(destination M.Socksaddr) M.Socksaddr {
	port := m.port
	if port == 0 && m.startPort != 0 && m.endPort != 0 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		port = uint16(r.Intn(int(m.endPort-m.startPort)+1) + int(m.startPort))
	}
	if m.prefix != nil {
		if m.prefix.Addr().Is4() {
			slice := m.prefix.Addr().As4()
			n := 1<<5 - m.prefix.Bits()
			if n == 0 {
				destination.Addr = m.prefix.Addr()
				if port != 0 {
					destination.Port = port
				}
				return destination
			}
			w1 := uint(n) / uint(1<<3)
			w2 := uint(n) % uint(1<<3)
			if w2 == 0 {
				w2 = 1 << 3
				w1 -= 1
			}
			for i := uint(0); i <= w1; i++ {
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				b := byte(r.Intn(1 << w2))
				if b == 0 {
					b = 1
				}
				slice[3-i] += b
			}
			destination.Addr = netip.AddrFrom4(slice)
			if port != 0 {
				destination.Port = port
			}
			return destination
		} else {
			slice := m.prefix.Addr().As16()
			n := 1<<7 - m.prefix.Bits()
			if n == 0 {
				destination.Addr = m.prefix.Addr()
				if port != 0 {
					destination.Port = port
				}
				return destination
			}
			w1 := uint(n) / uint(1<<3)
			w2 := uint(n) % uint(1<<3)
			if w2 == 0 {
				w2 = 1 << 3
				w1 -= 1
			}
			for i := uint(0); i <= w1; i++ {
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				b := byte(r.Intn(1 << w2))
				if b == 0 {
					b = 1
				}
				slice[15-i] += b
			}
			destination.Addr = netip.AddrFrom16(slice)
			if port != 0 {
				destination.Port = port
			}
			return destination
		}
	}
	if m.ip != nil {
		destination.Addr = *m.ip
	}
	if port != 0 {
		destination.Port = port
	}
	return destination
}
