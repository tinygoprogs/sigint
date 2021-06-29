package wifi

import (
	"context"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"log"
	"net"
	"runtime"
	"sync"
	"time"
)

const LogAccountingEveryDefault = time.Minute * 1
const DevChannelWidthDefault = 0x1000

type WifiConfig struct {
	Interface          string
	LogAccountingEvery time.Duration
	Handle             *pcap.Handle
	DevChannelWidth    int
}

type Wifi struct {
	WifiConfig
	cancel func()
	ch     chan *Device
	stats  *PacketStats
}

func NewWifi(conf WifiConfig) *Wifi {
	if conf.DevChannelWidth == 0 {
		conf.DevChannelWidth = DevChannelWidthDefault
	}
	if conf.LogAccountingEvery == 0 {
		conf.LogAccountingEvery = LogAccountingEveryDefault
	}
	return &Wifi{
		WifiConfig: conf,
		ch:         make(chan *Device, conf.DevChannelWidth),
		stats:      NewPacketStats("total", "interesting"),
	}
}

type InterestingLayers struct {
	RT    *layers.RadioTap
	Dot11 *layers.Dot11
	IEs   []*layers.Dot11InformationElement
	Stamp time.Time
}

// filter boring stuff like beacons, i don't care about the routers of this
// world
func (ils *InterestingLayers) filter() bool {
	if ils.Dot11.Type == layers.Dot11TypeMgmtBeacon {
		return true
	}
	return false
}

// may return nil, if no device intformation is found (i.e. certain multicast packets)
func NewInterestingLayers(p gopacket.Packet) (ils *InterestingLayers) {
	dot11layer := p.Layer(layers.LayerTypeDot11)
	if dot11layer == nil {
		return
	}
	dot11, ok := dot11layer.(*layers.Dot11)
	if !ok {
		return
	}
	phylayer := p.Layer(layers.LayerTypeRadioTap)
	if phylayer == nil {
		return
	}
	phy, ok := phylayer.(*layers.RadioTap)
	if !ok {
		return
	}
	ils = &InterestingLayers{
		RT:    phy,
		Dot11: dot11,
		Stamp: p.Metadata().CaptureInfo.Timestamp,
	}
	all := p.Layers()
	if len(all) >= 3 {
		ils.IEs = make([]*layers.Dot11InformationElement, 0, 10)
		for _, layer := range all[3:] {
			dot11i, ok := layer.(*layers.Dot11InformationElement)
			if ok && dot11i != nil {
				ils.IEs = append(ils.IEs, dot11i)
			}
		}
	}
	return
}

func (w *Wifi) logAccounting(ctx context.Context) {
	var ms runtime.MemStats
	log_stats := time.NewTicker(w.LogAccountingEvery)
	defer log_stats.Stop()
	logthings := func() {
		w.stats.log()
		runtime.ReadMemStats(&ms)
		log.Printf("MemStats[Alloc:%x]", ms.Alloc)
	}
	for {
		select {
		case <-log_stats.C:
			logthings()
		case <-ctx.Done():
			logthings()
			return
		}
	}
}

// Listen for packets in w.Handle, until ctx is Done() or the source is
// exausted
func (w *Wifi) Listen(ctx context.Context) {
	packets := gopacket.NewPacketSource(w.Handle, w.Handle.LinkType()).Packets()
	wg := sync.WaitGroup{}
	defer func() {
		w.cancel() // ensure that our context is canceled
		wg.Wait()  // wait for running stuff
		close(w.ch)
		log.Print("done")
	}()
	for {
		select {
		case packet, ok := <-packets:
			if !ok {
				log.Print("packet source closed")
				return
			}
			w.stats.inc("total")
			ils := NewInterestingLayers(packet)
			if ils == nil {
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				if ils.filter() {
					return
				}
				dev, addr := ils.ToDevice()
				if yes, reason := ShouldIgnore(addr); yes {
					log.Printf("skipping mac '%v' because %s", addr, reason)
					return
				}
				w.stats.inc("interesting")
				w.pushDevice(dev)
			}()
		case <-ctx.Done():
			log.Print("context done")
			return
		}
	}
}

func (ils *InterestingLayers) ToDevice() (*Device, net.HardwareAddr) {
	var addr net.HardwareAddr
	var da, sa, bssid, ta, ra net.HardwareAddr
	dot11 := ils.Dot11
	flags := ils.Dot11.Flags

	// DS -- Distribution System, but I want stations not AP's!
	if !flags.FromDS() && !flags.ToDS() {
		// who is the sender here? we only have one signal..
		da, sa, bssid = dot11.Address1, dot11.Address2, dot11.Address3
		fmt := "IBSS: currently unsupported, should return 2 deivces, da=%v, sa=%v, bssid=%v"
		log.Printf(fmt, da, sa, bssid)
	} else if !flags.FromDS() && flags.ToDS() {
		bssid, sa, da = dot11.Address1, dot11.Address2, dot11.Address3
		addr = sa
	} else if flags.FromDS() && !flags.ToDS() {
		da, bssid, sa = dot11.Address1, dot11.Address2, dot11.Address3
		addr = da
	} else /*fromDS and toDS*/ {
		ra, ta, da, sa = dot11.Address1, dot11.Address2, dot11.Address3, dot11.Address4
		if ta != nil && sa == nil {
			sa = ta
		}
		if ra != nil && da == nil {
			da = ra
		}
		fmt := "WDS bridge somewhere, action required? (da=%v, sa=%v, ta=%v, bssid=%v, addr=%v)"
		log.Printf(fmt, da, sa, ta, bssid, addr)
	}
	//_ = bssid

	return &Device{
		MAC: addr.String(),
		DataPoints: []*DataPoint{
			&DataPoint{
				Signal:    uint32(ils.RT.DBMAntennaSignal),
				Frequency: uint32(ils.RT.ChannelFrequency),
				TimeStamp: uint64(ils.Stamp.UnixNano()),
				Location:  &Coordinates{},
			},
		},
	}, addr
}

func (w *Wifi) pushDevice(dev *Device) {
	select {
	case w.ch <- dev:
	default:
		log.Printf("device channel full! dropping dev: %v!", dev.MAC)
	}
}

// start collecting + channel hopping + accounting
func (w *Wifi) Start(ctx context.Context) chan *Device {
	ctx, w.cancel = context.WithCancel(ctx)
	go HopChannels(ctx, w.Interface)
	go w.Listen(ctx)
	go w.logAccounting(ctx)
	return w.ch
}
