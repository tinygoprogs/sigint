package wifi

import (
	"context"
	"github.com/google/gopacket/pcap"
	"io/ioutil"
	"log"
	"testing"
)

func init() {
	if !testing.Verbose() {
		log.SetOutput(ioutil.Discard)
	}
}

func TestIlsToDevice(t *testing.T) {
	hnd, err := pcap.OpenOffline("testdata/random-wifi.cap")
	if err != nil || hnd == nil {
		t.Logf("hnd=%v, err=%v", hnd, err)
		t.Fail()
		return
	}
	w := NewWifi(WifiConfig{
		Handle: hnd,
	})
	dch := w.Start(context.Background())
	ndevs := 0
	for dev := range dch {
		ndevs++
		t.Log(dev)
	}
	if ndevs > 171 {
		t.Errorf("filtering failed: ndevs > 171, ndevs=%d", ndevs)
	}
}
