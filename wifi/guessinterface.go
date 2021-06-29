package wifi

import (
	"errors"
	"github.com/vishvananda/netlink"
)

// order corresponds to an imaginary likelyhood of correctness
var IfaceGuesses = []string{"wlan", "wlp", "wlx", "w"}

func BestGuessWifiIface() (netlink.Link, error) {
	var links []netlink.Link
	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}
	for _, guess := range IfaceGuesses {
		for _, l := range links {
			attrs := l.Attrs()
			if len(attrs.Name) < len(guess) {
				continue
			}
			if attrs.Name[:len(guess)] == guess {
				return l, nil
			}
		}
	}
	return nil, errors.New("no device found")
}
