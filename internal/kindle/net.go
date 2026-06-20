// Package kindle holds platform helpers specific to running on the Kindle:
// network/firewall, rootfs remounts, eips screen geometry, and the serial-
// prefix model table. Nothing in here knows about wallpapers or jsrepl.
package kindle

import (
	"log"
	"net"
	"os/exec"
	"strconv"
)

// DetectIP returns the first non-loopback IPv4 address on an up interface,
// or "" if none is available (e.g. wifi is off).
func DetectIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}
			if v4 := ipnet.IP.To4(); v4 != nil {
				return v4.String()
			}
		}
	}
	return ""
}

// OpenFirewall punches a hole in iptables for the given TCP port. Kindle's
// default INPUT policy drops inbound TCP, so phones on the LAN can't reach
// us without this. Idempotent: -C checks for the rule first.
func OpenFirewall(port int) {
	dport := strconv.Itoa(port)
	if exec.Command("iptables", "-C", "INPUT", "-p", "tcp", "--dport", dport, "-j", "ACCEPT").Run() == nil {
		return
	}
	if err := exec.Command("iptables", "-I", "INPUT", "-p", "tcp", "--dport", dport, "-j", "ACCEPT").Run(); err != nil {
		log.Printf("iptables open %s failed: %v", dport, err)
	}
}
