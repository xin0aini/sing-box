//go:build with_sideload

package outbound

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"sync/atomic"
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
	options         option.SideLoadOutboundOptions
	command         atomic.Pointer[exec.Cmd]
	isClose         atomic.Bool
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
		outbound.dialerForwarder = D.NewDialerForwarder(ctx, logger, outbound.dialer, options.ListenPort, M.ParseSocksaddrHostPort(options.Server, options.ServerPort), options.ListenNetwork.Build(), options.TCPFastOpen, options.UDPFragment, time.Duration(options.UDPTimeout)*time.Second)
	}
	serverSocksAddr := M.ParseSocksaddrHostPort("127.0.0.1", options.Socks5ProxyPort)
	outbound.socksClient = socks.NewClient(N.SystemDialer, serverSocksAddr, socks.Version5, "", "")
	command := exec.CommandContext(ctx, options.Command[0], options.Command[1:]...)
	command.Env = options.Env
	outbound.command.Store(command)
	outbound.options = options
	return outbound, nil
}

func (s *SideLoad) Start() error {
	if s.dialerForwarder != nil {
		err := s.dialerForwarder.Start()
		if err != nil {
			return err
		}
	}
	s.command.Load().Stdout = newSideLoadLogWriter(s.logger.Info)
	s.command.Load().Stderr = newSideLoadLogWriter(s.logger.Info)
	err := s.command.Load().Start()
	if err != nil {
		return err
	}
	go s.keepCommand()
	return nil
}

func (s *SideLoad) Close() error {
	s.isClose.Store(true)
	err := s.command.Load().Process.Kill()
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

func (s *SideLoad) keepCommand() {
	for {
		waitCtx, waitCancel := context.WithCancel(s.ctx)
		go func() {
			defer waitCancel()
			s.command.Load().Wait()
		}()
		select {
		case <-waitCtx.Done():
			if command := s.command.Load(); command != nil && command.ProcessState != nil && command.ProcessState.Exited() {
				if s.isClose.Load() {
					return
				}
				s.logger.Error("command stop, restart...")
				var newCommand *exec.Cmd
				for {
					select {
					case <-time.After(3 * time.Second):
						newCommand = exec.CommandContext(s.ctx, s.options.Command[0], s.options.Command[1:]...)
						newCommand.Env = s.options.Env
						newCommand.Stdout = newSideLoadLogWriter(s.logger.Info)
						newCommand.Stderr = newSideLoadLogWriter(s.logger.Info)
						err := newCommand.Start()
						if err != nil {
							newCommand.Process.Kill()
							s.logger.Error("restart command error: ", err, ", retry")
							continue
						}
					case <-s.ctx.Done():
						return
					}
					break
				}
				s.command.Store(newCommand)
				s.logger.Info("restart command success")
			}
		case <-s.ctx.Done():
			return
		}
	}
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
