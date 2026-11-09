package main

import "testing"

func TestVPNPathIdentification(t *testing.T) {
	networks := []byte("lan br-lan lan\nremote wg0 wireguard\nremote2 tun0 vpn\n")
	for _, tc := range []struct{ route, label, iface string }{
		{"192.0.2.5 dev wg0 src 192.0.2.1", "WireGuard VPN", "wg0"},
		{"2001:db8::5 dev tun0 src 2001:db8::1", "VPN connection", "tun0"},
		{"192.0.2.5 dev br-lan", "", ""},
		{"192.0.2.5 dev unknown0", "", ""},
		{"unreachable 192.0.2.5", "", ""},
	} {
		c := vpnConnection([]byte(tc.route), networks)
		if c.Path != tc.label || c.Interface != tc.iface || c.Radio != "" || c.Channel != 0 {
			t.Fatalf("route %q: %#v", tc.route, c)
		}
	}
}
