package serve

import (
	"net"
	"testing"
)

func ifs(names ...string) []net.Interface {
	var out []net.Interface
	for i, n := range names {
		out = append(out, net.Interface{Index: i + 1, Name: n, Flags: net.FlagUp})
	}
	return out
}

func addrs(m map[string]string) func(net.Interface) ([]net.Addr, error) {
	return func(i net.Interface) ([]net.Addr, error) {
		s, ok := m[i.Name]
		if !ok {
			return nil, nil
		}
		ip, n, _ := net.ParseCIDR(s)
		n.IP = ip
		return []net.Addr{n}, nil
	}
}

func TestFailsClosedWithNoTailscale(t *testing.T) {
	_, err := FindTailscale(ifs("en0", "lo0"), addrs(map[string]string{
		"en0": "192.168.1.40/24", "lo0": "127.0.0.1/8",
	}))
	if err != ErrNoTailscale {
		t.Fatalf("must refuse when tailscale is absent, got %v", err)
	}
}

func TestNeverFallsBackToLANOrLoopback(t *testing.T) {
	for name, cidr := range map[string]string{
		"en0": "192.168.1.40/24", "lo0": "127.0.0.1/8", "en1": "10.0.0.5/8", "utun9": "10.9.9.9/8",
	} {
		if _, err := FindTailscale(ifs(name), addrs(map[string]string{name: cidr})); err != ErrNoTailscale {
			t.Errorf("%s (%s) must NOT be accepted as a tailscale bind, got %v", name, cidr, err)
		}
	}
}

func TestAcceptsOnlyCGNATOnATunnel(t *testing.T) {
	a, err := FindTailscale(ifs("utun4"), addrs(map[string]string{"utun4": "100.101.102.103/32"}))
	if err != nil {
		t.Fatalf("a real tailscale address must be accepted: %v", err)
	}
	bind, err := SafeBindAddr(a, 2222)
	if err != nil || bind != "100.101.102.103:2222" {
		t.Fatalf("got %q %v", bind, err)
	}
}

func TestSafeBindRefusesWildcard(t *testing.T) {
	if _, err := SafeBindAddr(Addr{IP: net.ParseIP("0.0.0.0")}, 2222); err != ErrUnsafeBind {
		t.Fatal("0.0.0.0 must be refused — failing open is the whole risk")
	}
	if _, err := SafeBindAddr(Addr{}, 2222); err != ErrUnsafeBind {
		t.Fatal("an empty address must be refused, never defaulted")
	}
}
