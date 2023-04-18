package script

import (
	"context"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"os/exec"
	"time"
)

type ScriptService struct {
	options option.ScriptOptions
	ctx     context.Context
	logger  log.ContextLogger
	cmd     *exec.Cmd
	cancel  context.CancelFunc
}

func NewScript(ctx context.Context, logger log.ContextLogger, options option.ScriptOptions) *ScriptService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ScriptService{
		options: options,
		logger:  logger,
		ctx:     ctx,
	}
}

func (s *ScriptService) GetTag() string {
	return s.options.Tag
}

func (s *ScriptService) GetMode() string {
	return s.options.Mode
}

func (s *ScriptService) GetKeep() bool {
	return s.options.Keep
}

func (s *ScriptService) Start() error {
	ctx, cancel := context.WithCancel(s.ctx)
	s.cancel = cancel
	if s.options.Timeout > 0 {
		runCtx, runCancel := context.WithTimeout(ctx, time.Duration(s.options.Timeout))
		s.cancel = runCancel
		ctx = runCtx
	}
	s.cmd = exec.CommandContext(ctx, s.options.Script[0], s.options.Script[1:]...)
	if s.options.Output {
		s.cmd.Stdout = &logWriter{loggerFunc: s.logger.Info}
		s.cmd.Stderr = &logWriter{loggerFunc: s.logger.Error}
	}
	err := s.cmd.Start()
	if err != nil {
		if !s.options.IgnoreFail {
			return E.Cause(err, "script run failed")
		} else {
			s.logger.Error("script run failed: ", err.Error())
		}
	}
	return nil
}

func (s *ScriptService) Close() error {
	s.cancel()
	err := s.cmd.Wait()
	if err != nil {
		switch err {
		case context.Canceled:
		case context.DeadlineExceeded:
		default:
			return E.Cause(err, "script stop failed")
		}
	}
	return nil
}

func (s *ScriptService) RunWithGlobalContext() error {
	return s.RunWithContext(s.ctx)
}

func (s *ScriptService) RunWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if s.options.Timeout > 0 {
		runCtx, runCancel := context.WithTimeout(ctx, time.Duration(s.options.Timeout))
		defer runCancel()
		ctx = runCtx
	}
	s.cmd = exec.CommandContext(ctx, s.options.Script[0], s.options.Script[1:]...)
	if s.options.Output {
		s.cmd.Stdout = &logWriter{loggerFunc: s.logger.Info}
		s.cmd.Stderr = &logWriter{loggerFunc: s.logger.Error}
	}
	err := s.cmd.Run()
	if err != nil {
		if !s.options.IgnoreFail {
			return E.Cause(err, "script run failed")
		} else {
			s.logger.Error("script run failed: ", err.Error())
		}
	}
	return nil
}
