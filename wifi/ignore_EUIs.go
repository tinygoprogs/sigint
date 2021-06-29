package wifi

import (
	"net"
	"strings"
)

var IgnoreEUIs = []IgnoreEUIReason{
	{"zero", [2]string{"00:00:00:00:00:00", ""}},
	{"beacon", [2]string{"ff:ff:ff:ff:ff:ff", ""}},
	{"unicast prefix", [2]string{"00:00:5e:00:00:00", "00:00:5e:ff:ff:ff"}},
	{"multicast prefix", [2]string{"01:00:5e:00:00:00", "01:00:5e:ff:ff:ff"}},
	{"ipv6 multicast", [2]string{"33:33:00:00:00:00", "33:33:ff:ff:ff:ff"}},
}

type IgnoreEUIReason struct {
	Reason    string
	AddrRange [2]string
}

func (ireason *IgnoreEUIReason) Matches(addr string) bool {
	if ireason.AddrRange[1] == "" {
		return addr == ireason.AddrRange[0]
	}
	// hax
	idx := strings.Index(ireason.AddrRange[1], "ff")
	return strings.HasPrefix(addr, ireason.AddrRange[0][:idx])
}

func ShouldIgnore(addr net.HardwareAddr) (bool, string) {
	if addr == nil {
		return true, "nil"
	}
	for _, reason := range IgnoreEUIs {
		if reason.Matches(addr.String()) {
			return true, reason.Reason
		}
	}
	return false, ""
}
