//go:build with_sideload

package outbound

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/dialer"
	D "github.com/sagernet/sing-box/common/dialerforwarder"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/protocol/socks"
)

var _ adapter.Outbound = (*SideLoad)(nil)

type SideLoad struct {
	myOutboundAdapter
	ctx             context.Context
	dialer          N.Dialer
	socksClient     *socks.Client
	dialerForwarder *D.DialerForwarder
	command         *exec.Cmd
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
		outbound.dialerForwarder = D.NewDialerForwarder(ctx, logger, outbound.dialer, options.ListenPort, options.ServerOptions.Build(), options.ListenNetwork.Build(), options.TCPFastOpen, options.UDPFragment, time.Duration(options.UDPTimeout)*time.Second)
	}
	serverSocksAddr := M.ParseSocksaddrHostPort("127.0.0.1", options.Socks5ProxyPort)
	outbound.socksClient = socks.NewClient(N.SystemDialer, serverSocksAddr, socks.Version5, "", "")
	outbound.command = exec.CommandContext(ctx, options.Command[0], options.Command[1:]...)
	outbound.command.Env = options.Env
	return outbound, nil
}

func (s *SideLoad) Start() error {
	if s.dialerForwarder != nil {
		err := s.dialerForwarder.Start()
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
	return nil
}

func (s *SideLoad) Close() error {
	err := s.command.Process.Kill()
	if err != nil {
		return err
	}
	if s.dialerForwarder != nil {
		err = s.dialerForwarder.Close()
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
