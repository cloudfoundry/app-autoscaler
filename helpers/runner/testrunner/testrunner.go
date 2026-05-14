package testrunner

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/runner"
)

type Process struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error
	mu     sync.Mutex
}

func Invoke(r runner.Runner) *Process {
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	p := &Process{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		defer close(p.done)
		err := r.Run(ctx, ready)
		p.mu.Lock()
		p.err = err
		p.mu.Unlock()
	}()

	select {
	case <-ready:
	case <-p.done:
	}

	return p
}

func (p *Process) Interrupt() {
	p.cancel()
}

func (p *Process) Kill(timeout time.Duration) {
	p.cancel()
	select {
	case <-p.done:
	case <-time.After(timeout):
	}
}

func (p *Process) Wait() <-chan struct{} {
	return p.done
}

func (p *Process) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

type CmdConfig struct {
	Name              string
	Command           *exec.Cmd
	StartCheck        string
	StartCheckTimeout time.Duration
}

type cmdRunner struct {
	cfg CmdConfig
}

func NewCmdRunner(cfg CmdConfig) runner.Runner {
	if cfg.StartCheckTimeout == 0 {
		cfg.StartCheckTimeout = 20 * time.Second
	}
	return &cmdRunner{cfg: cfg}
}

func (r *cmdRunner) Run(ctx context.Context, ready chan<- struct{}) error {
	cmd := r.cfg.Command
	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("cmd runner %s: pipe: %w", r.cfg.Name, err)
	}
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return fmt.Errorf("cmd runner %s: start: %w", r.cfg.Name, err)
	}
	_ = pw.Close()

	if err := r.waitForStartCheck(ctx, pr, ready); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-done
		return nil
	case err := <-done:
		return err
	}
}

func (r *cmdRunner) waitForStartCheck(ctx context.Context, pr *os.File, ready chan<- struct{}) error {
	if r.cfg.StartCheck == "" {
		_ = pr.Close()
		close(ready)
		return nil
	}

	startCheckReady := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), r.cfg.StartCheck) {
				close(startCheckReady)
				for scanner.Scan() {
				}
				_ = pr.Close()
				return
			}
		}
		_ = pr.Close()
	}()

	select {
	case <-startCheckReady:
		close(ready)
		return nil
	case <-time.After(r.cfg.StartCheckTimeout):
		return fmt.Errorf("cmd runner %s: start check %q not seen within %v", r.cfg.Name, r.cfg.StartCheck, r.cfg.StartCheckTimeout)
	case <-ctx.Done():
		return ctx.Err()
	}
}
