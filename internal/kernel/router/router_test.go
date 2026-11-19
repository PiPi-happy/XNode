package router

import (
	"testing"

	"github.com/cedar2025/xboard-node/internal/config"
)

func TestRouterAddsMieruAndSelectsMBoxSingBox(t *testing.T) {
	r := New(config.KernelConfig{Type: "singbox"})
	if r.kernelFor("mieru").Name() != "sing-box" {
		t.Fatalf("Mieru backend = %q, want sing-box", r.kernelFor("mieru").Name())
	}
	if r.kernelFor("vless").Name() != "sing-box" {
		t.Fatalf("VLESS backend = %q, want sing-box", r.kernelFor("vless").Name())
	}
	found := false
	for _, protocol := range r.Protocols() {
		if protocol == "mieru" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Protocols() does not include Mieru: %v", r.Protocols())
	}
}

func TestRouterKeepsXrayForOtherProtocols(t *testing.T) {
	r := New(config.KernelConfig{Type: "xray"})
	if got := r.kernelFor("mieru").Name(); got != "sing-box" {
		t.Fatalf("Mieru backend = %q, want sing-box", got)
	}
	if got := r.kernelFor("vmess").Name(); got != "xray" {
		t.Fatalf("VMess backend = %q, want xray", got)
	}
}
