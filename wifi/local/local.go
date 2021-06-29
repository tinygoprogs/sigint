package local

import (
	"context"
	"github.com/tinygoprogs/sigint/wifi"
)

type Config struct {
	Local bool
	LConf LocalConfig
	Wifi  wifi.WifiConfig
}

// Collect forever, unless an error occurs.
func Collect(ctx context.Context, conf Config) error {
	ls, err := NewLStore(ctx, &conf.LConf)
	if err != nil {
		return err
	}
	src := wifi.NewWifi(conf.Wifi)
	devices := src.Start(ctx)
	for dev := range devices {
		devs := wifi.Devices{Devices: []*wifi.Device{dev}}
		ls.NewDevices(ctx, &devs)
	}
	return nil
}
