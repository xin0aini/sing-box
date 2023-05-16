//go:build with_sideload

package outbound

import (
	"context"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/dialer"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/common/udpnat"
	"github.com/sagernet/sing/protocol/socks"
)

var _ adapter.Outbound = (*SideLoad)(nil)

type SideLoad struct {
	myOutboundAdapter
	ctx         context.Context
	dialer      N.Dialer
	socksClient *socks.Client
	sll         *sideLoadListener
	command     *exec.Cmd
}

func NewSideLoad(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.SideLoadOutboundOptions) (*SideLoad, error) {
	outbound := &SideLoad{
		myOutboundAdapter: myOutboundAdapter{
			protocol: C.TypeSideLoad,
			network:  options.Network.Build(),
			router:   router,
			logger:   logger,
			tag:      tag,
		},
		ctx:    ctx,
		dialer: dialer.New(router, options.DialerOptions),
	}
	if options.Command == nil || len(options.Command) == 0 {
		return nil, E.New("command not found")
	}
	if options.Socks5ProxyPort == 0 {
		return nil, E.New("socks5 proxy port not found")
	}
	if options.ListenPort != 0 && options.Server != "" && options.ServerPort != 0 {
		outbound.sll = &sideLoadListener{
			outbound:    outbound,
			port:        options.ListenPort,
			network:     options.ListenNetwork.Build(),
			destination: options.ServerOptions.Build(),
		}
	}
	serverSocksAddr := M.ParseSocksaddrHostPort("127.0.0.1", options.Socks5ProxyPort)
	outbound.socksClient = socks.NewClient(N.SystemDialer, serverSocksAddr, socks.Version5, "", "")
	outbound.command = exec.CommandContext(ctx, options.Command[0], options.Command[1:]...)
	outbound.command.Env = options.Env
	return outbound, nil
}

func (s *SideLoad) Start() error {
	if s.sll != nil {
		err := s.sll.Start()
		if err != nil {
			return err
		}
	}
	s.command.Stdout = newSideLoadLogWriter(s.logger.Info)
	s.command.Stderr = newSideLoadLogWriter(s.logger.Info)
	err := s.command.Start()
	if err != nil {
		return err
	}
	s.logger.Info("command start")
	return nil
}

func (s *SideLoad) Close() error {
	err := s.command.Process.Kill()
	if err != nil {
		return err
	}
	s.logger.Info("command stop")
	if s.sll != nil {
		err = s.sll.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SideLoad) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	s.logger.InfoContext(ctx, "outbound connection to ", destination)
	return s.socksClient.DialContext(ctx, network, destination)
}

func (s *SideLoad) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	ctx, metadata := adapter.AppendContext(ctx)
	metadata.Outbound = s.tag
	metadata.Destination = destination
	s.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	return s.socksClient.ListenPacket(ctx, destination)
}

func (s *SideLoad) NewConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext) error {
	return NewConnection(ctx, s, conn, metadata)
}

func (s *SideLoad) NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext) error {
	return NewPacketConnection(ctx, s, conn, metadata)
}

type sideLoadLogWriter struct {
	f func(a ...any)
}

func newSideLoadLogWriter(logFunc func(a ...any)) *sideLoadLogWriter {
	return &sideLoadLogWriter{f: logFunc}
}

func (s *sideLoadLogWriter) Write(p []byte) (int, error) {
	ps := strings.Split(string(p), "\n")
	for _, p := range ps {
		if len(p) == 0 {
			continue
		}
		s.f(p)
	}
	return len(p), nil
}

type sideLoadDNSCache struct {
	domain     string
	addrs      []netip.Addr
	lookupTime time.Time
}

type sideLoadListener struct {
	outbound *SideLoad
	port     uint16
	network  []string

	destination          M.Socksaddr
	dstDNSCache          atomic.Value
	tcpListener          net.Listener
	udpConn              *net.UDPConn
	udpNat               *udpnat.Service[netip.AddrPort]
	packetOutboundClosed chan struct{}
	packetOutbound       chan *sideLoadPacket
	inShutdown           atomic.Bool
}

func (s *sideLoadListener) Start() error {
	s.packetOutboundClosed = make(chan struct{})
	s.packetOutbound = make(chan *sideLoadPacket)
	s.udpNat = udpnat.New[netip.AddrPort](int64(C.UDPTimeout.Seconds()), s)
	if common.Contains(s.network, N.NetworkTCP) {
		var err error
		s.tcpListener, err = net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: int(s.port)})
		if err != nil {
			return err
		}
		s.outbound.logger.Info("tcp server started at ", s.tcpListener.Addr())
		go s.dialTCP()
	}
	if common.Contains(s.network, N.NetworkUDP) {
		var err error
		s.udpConn, err = net.ListenUDP(N.NetworkUDP, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: int(s.port)})
		if err != nil {
			return err
		}
		s.outbound.logger.Info("udp server started at ", s.udpConn.LocalAddr())
		go s.dialUDP()
		go s.handleUDPOut()
	}
	return nil
}

func (s *sideLoadListener) Close() error {
	s.inShutdown.Store(true)
	if s.tcpListener != nil {
		err := s.tcpListener.Close()
		if err != nil {
			return err
		}
	}
	return common.Close(common.PtrOrNil(s.udpConn))
}

func (s *sideLoadListener) dnsLookup(destination *M.Socksaddr) bool {
	lookupDone := false
	c := s.dstDNSCache.Load()
	if c != nil {
		cache := c.(*sideLoadDNSCache)
		if time.Now().Sub(cache.lookupTime) < 5*time.Minute {
			destination.Addr = cache.addrs[0]
			lookupDone = true
		}
	}
	if !lookupDone {
		ctx, metadata := adapter.AppendContext(s.outbound.ctx)
		metadata.Outbound = s.outbound.tag
		addrs, err := s.outbound.router.LookupDefault(ctx, s.destination.Fqdn)
		if err != nil {
			s.outbound.logger.Error(E.Cause(err, "dns lookup failed"))
			return false
		}
		sc := &sideLoadDNSCache{
			domain:     destination.Fqdn,
			addrs:      addrs,
			lookupTime: time.Now(),
		}
		s.dstDNSCache.Store(sc)
		destination.Addr = addrs[0]
	}
	destination.Fqdn = ""
	return true
}

func (s *sideLoadListener) dialTCP() {
	for {
		conn, err := s.tcpListener.Accept()
		if err != nil {
			if netError, isNetError := err.(net.Error); isNetError && netError.Temporary() {
				s.outbound.logger.Error(err)
				continue
			}
			if s.inShutdown.Load() && E.IsClosed(err) {
				return
			}
			s.tcpListener.Close()
			s.outbound.logger.Error("serve error: ", err)
			continue
		}
		go s.handleTCP(conn)
	}
}

func (s *sideLoadListener) handleTCP(conn net.Conn) {
	defer conn.Close()
	destination := s.destination
	if destination.IsFqdn() {
		if !s.dnsLookup(&destination) {
			return
		}
	}
	outConn, err := s.outbound.dialer.DialContext(s.outbound.ctx, N.NetworkTCP, destination)
	if err != nil {
		s.outbound.logger.Error(E.Cause(err, "outbound connection failed"))
		return
	}
	defer outConn.Close()
	err = CopyEarlyConn(s.outbound.ctx, conn, outConn)
	if err != nil {
		s.outbound.logger.Error(err)
		return
	}
}

func (s *sideLoadListener) dialUDP() {
	defer close(s.packetOutboundClosed)
	for {
		buffer := buf.NewPacket()
		n, addr, err := s.udpConn.ReadFromUDPAddrPort(buffer.FreeBytes())
		if err != nil {
			buffer.Release()
			return
		}
		buffer.Truncate(n)
		err = s.handleUDP(buffer, addr)
		if err != nil {
			buffer.Release()
			s.outbound.logger.Error(E.Cause(err, "process packet from ", M.SocksaddrFromNetIP(addr).Unwrap()))
		}
	}
}

func (s *sideLoadListener) handleUDP(buffer *buf.Buffer, addr netip.AddrPort) error {
	destination := s.destination
	if destination.IsFqdn() {
		if !s.dnsLookup(&destination) {
			return E.New("dns lookup failed")
		}
	}
	metadata := M.Metadata{
		Source:      M.SocksaddrFromNetIP(addr).Unwrap(),
		Destination: destination,
	}
	s.udpNat.NewContextPacket(s.outbound.ctx, metadata.Source.AddrPort(), buffer, metadata, func(natConn N.PacketConn) (context.Context, N.PacketWriter) {
		return s.outbound.ctx, &udpnat.DirectBackWriter{Source: (*sideLoadPacketService)(s), Nat: natConn}
	})
	return nil
}

func (s *sideLoadListener) handleUDPOut() {
	for {
		select {
		case packet := <-s.packetOutbound:
			err := s.writePacket(packet.buffer, packet.destination)
			if err != nil && !E.IsClosed(err) {
				s.outbound.logger.Error(E.New("write back udp: ", err))
			}
			continue
		case <-s.packetOutboundClosed:
		}
		for {
			select {
			case packet := <-s.packetOutbound:
				packet.buffer.Release()
			default:
				return
			}
		}
	}
}

func (s *sideLoadListener) writePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	defer buffer.Release()
	return common.Error(s.udpConn.WriteToUDPAddrPort(buffer.Bytes(), destination.AddrPort()))
}

type sideLoadPacket struct {
	buffer      *buf.Buffer
	destination M.Socksaddr
}

type sideLoadPacketService sideLoadListener

func (s *sideLoadPacketService) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	n, addr, err := s.udpConn.ReadFromUDPAddrPort(buffer.FreeBytes())
	if err != nil {
		return M.Socksaddr{}, err
	}
	buffer.Truncate(n)
	return M.SocksaddrFromNetIP(addr), nil
}

func (s *sideLoadPacketService) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	select {
	case s.packetOutbound <- &sideLoadPacket{buffer, destination}:
		return nil
	case <-s.packetOutboundClosed:
		return os.ErrClosed
	}
}

func (s *sideLoadPacketService) LocalAddr() net.Addr {
	return s.udpConn.LocalAddr()
}

func (s *sideLoadPacketService) SetDeadline(t time.Time) error {
	return s.udpConn.SetDeadline(t)
}

func (s *sideLoadPacketService) SetReadDeadline(t time.Time) error {
	return s.udpConn.SetReadDeadline(t)
}

func (s *sideLoadPacketService) SetWriteDeadline(t time.Time) error {
	return s.udpConn.SetWriteDeadline(t)
}

func (s *sideLoadPacketService) Close() error {
	return s.udpConn.Close()
}

func (s *sideLoadListener) NewError(ctx context.Context, err error) {
	common.Close(err)
	if E.IsClosedOrCanceled(err) {
		s.outbound.logger.DebugContext(ctx, "connection closed: ", err)
		return
	}
	s.outbound.logger.ErrorContext(ctx, err)
}

func (s *sideLoadListener) NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata M.Metadata) error {
	outConn, err := s.outbound.dialer.ListenPacket(ctx, metadata.Destination)
	if err != nil {
		return N.HandshakeFailure(conn, err)
	}
	return bufio.CopyPacketConn(ctx, conn, bufio.NewPacketConn(outConn))
}
