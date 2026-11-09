package panel

import (
	"encoding/json"
	"testing"
)

func TestStringOrArray_UnmarshalString(t *testing.T) {
	input := `"hello world"`
	var s StringOrArray
	if err := json.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("unmarshal string: %v", err)
	}
	if string(s) != "hello world" {
		t.Errorf("got %q, want %q", s, "hello world")
	}
}

func TestStringOrArray_UnmarshalArray(t *testing.T) {
	input := `["stop=8","0=30-30","1=100-400"]`
	var s StringOrArray
	if err := json.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("unmarshal array: %v", err)
	}
	want := "stop=8\n0=30-30\n1=100-400"
	if string(s) != want {
		t.Errorf("got %q, want %q", s, want)
	}
}

func TestStringOrArray_UnmarshalEmptyArray(t *testing.T) {
	input := `[]`
	var s StringOrArray
	if err := json.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("unmarshal empty array: %v", err)
	}
	if string(s) != "" {
		t.Errorf("got %q, want empty", s)
	}
}

func TestStringOrArray_UnmarshalNull(t *testing.T) {
	input := `null`
	var s StringOrArray
	if err := json.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if string(s) != "" {
		t.Errorf("got %q, want empty", s)
	}
}

func TestNodeConfig_UnmarshalWithPaddingSchemeArray(t *testing.T) {
	input := `{
		"protocol": "anytls",
		"server_port": 443,
		"padding_scheme": ["stop=8", "0=30-30", "1=100-400"]
	}`
	var nc NodeConfig
	if err := json.Unmarshal([]byte(input), &nc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if nc.Protocol != "anytls" {
		t.Errorf("protocol: got %q, want %q", nc.Protocol, "anytls")
	}
	want := "stop=8\n0=30-30\n1=100-400"
	if string(nc.PaddingScheme) != want {
		t.Errorf("padding_scheme: got %q, want %q", nc.PaddingScheme, want)
	}
}

func TestNodeConfig_UnmarshalWithPaddingSchemeString(t *testing.T) {
	input := `{
		"protocol": "anytls",
		"server_port": 443,
		"padding_scheme": "stop=8\n0=30-30"
	}`
	var nc NodeConfig
	if err := json.Unmarshal([]byte(input), &nc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := "stop=8\n0=30-30"
	if string(nc.PaddingScheme) != want {
		t.Errorf("padding_scheme: got %q, want %q", nc.PaddingScheme, want)
	}
}

func TestNodeConfig_UnmarshalFullPanelResponse(t *testing.T) {
	input := `{
		"protocol": "shadowsocks",
		"listen_ip": "0.0.0.0",
		"server_port": 111,
		"cipher": "aes-128-gcm",
		"network": null,
		"networkSettings": null,
		"server_name": "",
		"base_config": {
			"push_interval": 60,
			"pull_interval": 60
		}
	}`
	var nc NodeConfig
	if err := json.Unmarshal([]byte(input), &nc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if nc.Protocol != "shadowsocks" {
		t.Errorf("protocol: got %q", nc.Protocol)
	}
	if nc.ServerPort != 111 {
		t.Errorf("server_port: got %d", nc.ServerPort)
	}
	if nc.Cipher != "aes-128-gcm" {
		t.Errorf("cipher: got %q", nc.Cipher)
	}
	if nc.BaseConfig.PushInterval != 60 {
		t.Errorf("push_interval: got %d", nc.BaseConfig.PushInterval)
	}
}

func TestNodeConfig_UnmarshalXboardExtendedConfigShape(t *testing.T) {
	input := `{
		"protocol": "vless",
		"listen_ip": "0.0.0.0",
		"server_port": 443,
		"network": "tcp",
		"networkSettings": {"acceptProxyProtocol": true},
		"tls": 1,
		"flow": "xtls-rprx-vision",
		"decryption": "none",
		"tls_settings": {"server_name": "node.example.com"},
		"routes": [{"id": 7, "match": ["geosite:cn"], "action": "block"}],
		"custom_outbounds": [{"tag": "direct-alt", "protocol": "freedom", "settings": {"domainStrategy": "UseIPv4"}}],
		"custom_routes": [{"type": "field", "outboundTag": "direct-alt"}],
		"custom_route_rules": [{"name": "local domains", "match": {"domain_suffixes": ["example.cn"]}, "action": {"type": "outbound", "target": "direct-alt"}}],
		"cert_config": {"mode": "file", "domain": "node.example.com", "cert_file": "/etc/ssl/cert.pem", "key_file": "/etc/ssl/key.pem"},
		"multiplex": {"enabled": true, "protocol": "smux", "max_connections": 4, "min_streams": 2, "max_streams": 8, "padding": true, "brutal": {"enabled": true, "up_mbps": 100, "down_mbps": 300}}
	}`

	var nc NodeConfig
	if err := json.Unmarshal([]byte(input), &nc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if nc.Protocol != "vless" || nc.ListenIP != "0.0.0.0" || nc.ServerPort != 443 || nc.Network != "tcp" {
		t.Fatalf("base node fields not decoded: %+v", nc)
	}
	if !nc.GetProxyProtocol() || nc.TLS != 1 || nc.Flow != "xtls-rprx-vision" || nc.TLSSettings["server_name"] != "node.example.com" {
		t.Fatalf("protocol/network settings not decoded: %+v", nc)
	}
	if len(nc.Routes) != 1 || len(nc.CustomOutbounds) != 1 || len(nc.CustomRoutes) != 1 || len(nc.CustomRouteRules) != 1 {
		t.Fatalf("route and outbound extensions not decoded: routes=%d outbounds=%d custom_routes=%d custom_route_rules=%d", len(nc.Routes), len(nc.CustomOutbounds), len(nc.CustomRoutes), len(nc.CustomRouteRules))
	}
	if nc.CertConfig == nil || nc.CertConfig.CertMode != "file" || nc.CertConfig.CertFile != "/etc/ssl/cert.pem" {
		t.Fatalf("certificate config not decoded: %+v", nc.CertConfig)
	}
	if nc.Multiplex == nil || nc.Multiplex.Brutal == nil || !nc.Multiplex.Brutal.Enabled || nc.Multiplex.Brutal.DownMbps != 300 {
		t.Fatalf("multiplex config not decoded: %+v", nc.Multiplex)
	}
}

func TestUsersResponse_Unmarshal(t *testing.T) {
	input := `{
		"users": [
			{"id": 1, "uuid": "abc-123", "speed_limit": 3, "device_limit": 2},
			{"id": 2, "uuid": "def-456", "speed_limit": 0, "device_limit": 0}
		]
	}`
	var resp UsersResponse
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Users) != 2 {
		t.Fatalf("users count: got %d, want 2", len(resp.Users))
	}
	if resp.Users[0].SpeedLimit != 3 {
		t.Errorf("user[0].speed_limit: got %d", resp.Users[0].SpeedLimit)
	}
	if resp.Users[1].DeviceLimit != 0 {
		t.Errorf("user[1].device_limit: got %d", resp.Users[1].DeviceLimit)
	}
}
