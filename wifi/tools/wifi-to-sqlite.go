package main

import (
	"context"
	"flag"
	"github.com/google/gopacket/pcap"
	"github.com/tinygoprogs/sigint/wifi"
	"github.com/tinygoprogs/sigint/wifi/local"
	"log"
	"os"
	"os/signal"
	"time"
)

var (
	FTimeout = flag.Int("timeout", 0, "collect only for this amount of seconds")
	FDbname  = flag.String("dbname", "test.db", "name of the sqlite db")
	FIface   = flag.String("interface", "", "default is to guess")
)

func init() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
}

func main() {
	var err error
	cnf := local.Config{
		Local: true,
		LConf: local.LocalConfig{
			File:     *FDbname,
			Nmaps:    20,
			ChanSize: 0x100,
		},
		Wifi: wifi.WifiConfig{
			LogAccountingEvery: time.Minute * 1,
		},
	}

	var ifname string
	if *FIface != "" {
		ifname = *FIface
	} else {
		ifa, err := wifi.BestGuessWifiIface()
		if err != nil {
			log.Fatal(err)
		}
		ifname = ifa.Attrs().Name
	}
	cnf.Wifi.Interface = ifname
	cnf.Wifi.Handle, err = pcap.OpenLive(ifname, 1600, true, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	if *FTimeout != 0 {
		ctx, _ = context.WithTimeout(ctx, time.Second*time.Duration(*FTimeout))
	}
	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt)
		select {
		case <-ch:
			cancel()
		case <-ctx.Done():
			break
		}
	}()

	local.Collect(ctx, cnf)
}
