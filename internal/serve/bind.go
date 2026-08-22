package serve

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

var (
	ErrNoTailscale = errors.New("no tailscale interface found — refusing to start")
	ErrUnsafeBind  = errors.New("refusing to bind a non-tailscale address")
)

func isTailscaleCGNAT(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	return v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127
}

func looksLikeTailscaleIface(name string) bool {
	n := strings.ToLower(name)
	return strings.HasPrefix(n, "tailscale") || strings.HasPrefix(n, "utun") || strings.HasPrefix(n, "ts")
}

type Addr struct {
	Iface string
	IP    net.IP
}

func FindTailscale(ifaces []net.Interface, addrsOf func(net.Interface) ([]net.Addr, error)) (Addr, error) {
	for _, in := range ifaces {
		if in.Flags&net.FlagUp == 0 || !looksLikeTailscaleIface(in.Name) {
			continue
		}
		as, err := addrsOf(in)
		if err != nil {
			continue
		}
		for _, a := range as {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && isTailscaleCGNAT(ip) {
				return Addr{Iface: in.Name, IP: ip}, nil
			}
		}
	}
	return Addr{}, ErrNoTailscale
}

func SafeBindAddr(a Addr, port int) (string, error) {
	if a.IP == nil || !isTailscaleCGNAT(a.IP) {
		return "", ErrUnsafeBind
	}
	return fmt.Sprintf("%s:%d", a.IP.String(), port), nil
}

func Resolve(port int) (string, string, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return "", "", err
	}
	a, err := FindTailscale(ifs, func(i net.Interface) ([]net.Addr, error) { return i.Addrs() })
	if err != nil {
		return "", "", err
	}
	bind, err := SafeBindAddr(a, port)
	return bind, a.Iface, err
}
