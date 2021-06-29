package wifi

import (
	"net"
	"testing"
)

func TestIgnoreSomeEUIs(t *testing.T) {
	cases := map[string]bool{
		"ff:ff:ff:ff:ff:ff": true,
		"00:01:5e:00:00:00": false,
		"00:00:5e:01:00:00": true,
		"01:00:5e:01:00:00": true,
		"01:01:5e:01:00:00": false,
	}
	for eui, expected := range cases {
		addr, err := net.ParseMAC(eui)
		if err != nil {
			t.Fatalf("could not parse test case: '%v'", eui)
		}
		result, reason := ShouldIgnore(addr)
		if result != expected {
			t.Errorf("got '%v', expected '%v', reason: '%v'", result, expected, reason)
		}
	}
}
