// Package landiscovery advertises only private-network HTTPS server endpoints.
package landiscovery

import (
	"fmt"
	"github.com/grandcat/zeroconf"
	"net"
	"os"
	"strconv"
	"strings"
)

// Advertise does not treat discovery as trust or expose account/PIN information.
// Loopback-only listeners are deliberately not announced to other LAN devices.
func Advertise(address string) (func(), error) {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return func() {}, nil
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid discovery port")
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var private []net.Interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, _ := iface.Addrs()
		for _, addr := range addresses {
			ip, _, err := net.ParseCIDR(addr.String())
			if err == nil && ip.IsPrivate() && (host == "" || host == "0.0.0.0" || host == "::" || ip.String() == host) {
				private = append(private, iface)
				break
			}
		}
	}
	if len(private) == 0 {
		return func() {}, nil
	}
	name, _ := os.Hostname()
	name = "AlreadyOn " + strings.Split(name, ".")[0]
	if len(name) > 63 {
		name = name[:63]
	}
	server, err := zeroconf.Register(name, "_alreadyon._tcp", "local.", port, []string{"protocol=1", "scheme=https"}, private)
	if err != nil {
		return nil, err
	}
	return server.Shutdown, nil
}
