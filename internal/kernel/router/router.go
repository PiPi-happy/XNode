// Package router selects the configured proxy kernel, with MBox-backed
// sing-box for Mieru nodes.
package router

import (
	"context"
	"fmt"
	"sync"

	"github.com/cedar2025/xboard-node/internal/config"
	"github.com/cedar2025/xboard-node/internal/kernel"
	"github.com/cedar2025/xboard-node/internal/kernel/singbox"
	"github.com/cedar2025/xboard-node/internal/kernel/xray"
	"github.com/cedar2025/xboard-node/internal/model"
	"golang.org/x/time/rate"
)

// Router keeps the existing sing-box/Xray selection and routes Mieru nodes to
// the Mieru-enabled sing-box fork.
type Router struct {
	cfg config.KernelConfig

	mu       sync.RWMutex
	active   kernel.Kernel
	speedFn  func(string) *rate.Limiter
	deviceFn func(string) (int, bool)
	devices  map[int][]string
}

func New(cfg config.KernelConfig) *Router {
	return &Router{cfg: cfg}
}

func (r *Router) Name() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active != nil {
		return r.active.Name()
	}
	return r.defaultKernel().Name()
}

func (r *Router) Protocols() []string {
	protocols := append([]string(nil), r.defaultKernel().Protocols()...)
	for _, p := range protocols {
		if p == "mieru" {
			return protocols
		}
	}
	return append(protocols, "mieru")
}

func (r *Router) Capabilities() kernel.Capabilities {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active != nil {
		return r.active.Capabilities()
	}
	return r.defaultKernel().Capabilities()
}

func (r *Router) Start(nc *model.NodeSpec, users []model.UserSpec, tls kernel.TLSCert) error {
	if nc == nil {
		return fmt.Errorf("node config is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	next := r.kernelFor(nc.Protocol)
	if r.active != nil {
		if r.active.Name() != next.Name() {
			r.active.Stop()
		} else {
			// Reuse the live backend when the selected kernel type is unchanged.
			// kernelFor constructs a fresh wrapper, so replacing active here without
			// stopping it would leak the previous listeners and goroutines.
			next = r.active
		}
	}
	r.active = next
	r.configure(next)
	return next.Start(nc, users, tls)
}

func (r *Router) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != nil {
		r.active.Stop()
	}
}

func (r *Router) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active != nil && r.active.IsRunning()
}

func (r *Router) Reload(nc *model.NodeSpec, users []model.UserSpec, tls kernel.TLSCert) error {
	if nc == nil {
		return fmt.Errorf("node config is nil")
	}
	r.mu.RLock()
	active := r.active
	if active != nil && active.Name() == r.kernelFor(nc.Protocol).Name() {
		err := active.Reload(nc, users, tls)
		r.mu.RUnlock()
		return err
	}
	r.mu.RUnlock()
	return r.Start(nc, users, tls)
}

func (r *Router) AddUsers(users []model.UserSpec) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active == nil {
		return 0, fmt.Errorf("proxy kernel has not been started")
	}
	return r.active.AddUsers(users)
}

func (r *Router) RemoveUsers(users []model.UserSpec) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active == nil {
		return 0, fmt.Errorf("proxy kernel has not been started")
	}
	return r.active.RemoveUsers(users)
}

func (r *Router) UpdateUsers(users []model.UserSpec) (int, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active == nil {
		return 0, 0, fmt.Errorf("proxy kernel has not been started")
	}
	return r.active.UpdateUsers(users)
}

func (r *Router) GetUserTraffic(ctx context.Context) (map[int][2]int64, map[int]map[string]bool, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active == nil {
		return map[int][2]int64{}, map[int]map[string]bool{}, 0, nil
	}
	return r.active.GetUserTraffic(ctx)
}

func (r *Router) CloseConnection(ctx context.Context, connID string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active == nil {
		return fmt.Errorf("proxy kernel has not been started")
	}
	return r.active.CloseConnection(ctx, connID)
}

func (r *Router) CloseUserConnections(ctx context.Context, uuid string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active == nil {
		return fmt.Errorf("proxy kernel has not been started")
	}
	return r.active.CloseUserConnections(ctx, uuid)
}

func (r *Router) SetSpeedLimitFunc(fn func(string) *rate.Limiter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.speedFn = fn
	if r.active != nil {
		r.active.SetSpeedLimitFunc(fn)
	}
}

func (r *Router) SetDeviceLimitFunc(fn func(string) (int, bool)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deviceFn = fn
	if r.active != nil {
		r.active.SetDeviceLimitFunc(fn)
	}
}

func (r *Router) UpdateGlobalDevices(users map[int][]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices = users
	if r.active != nil {
		r.active.UpdateGlobalDevices(users)
	}
}

func (r *Router) ClearGlobalDevices() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices = nil
	if r.active != nil {
		r.active.ClearGlobalDevices()
	}
}

func (r *Router) kernelFor(protocol string) kernel.Kernel {
	if protocol == "mieru" {
		return singbox.New(r.cfg)
	}
	return r.defaultKernel()
}

func (r *Router) defaultKernel() kernel.Kernel {
	switch r.cfg.Type {
	case "xray":
		return xray.New(r.cfg)
	case "singbox", "sing-box", "":
		return singbox.New(r.cfg)
	default:
		return singbox.New(r.cfg)
	}
}

func (r *Router) configure(k kernel.Kernel) {
	k.SetSpeedLimitFunc(r.speedFn)
	k.SetDeviceLimitFunc(r.deviceFn)
	if r.devices != nil {
		k.UpdateGlobalDevices(r.devices)
	}
}

var _ kernel.Kernel = (*Router)(nil)
