package location

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"sync"
	"time"
)

type Config struct {
	UpdateInterval    time.Duration
	updateWaitTimeout time.Duration
}

type Location struct {
	Stamp time.Time
	Lat   float32
	Lon   float32
	Alt   float64
	Acc   float64
}

type Provider struct {
	Conf         Config
	m            sync.Mutex
	lastLocation *Location
}

func NewProvider(c Config) *Provider {
	c.updateWaitTimeout = c.UpdateInterval / 2
	return &Provider{
		Conf: c,
	}
}

/*
location format:
```
u0_a142:~$ termux-location
{
	"latitude": 48.77833406,
	"longitude": 11.43595288,
	"altitude": 434.0,
	"accuracy": 28.0,
	"vertical_accuracy": 0.0,
	"bearing": 0.0,
	"speed": 0.0,
	"elapsedMs": 12,
	"provider": "gps"
}
```
*/
type termuxLocation struct {
	Latitude          float32
	Longitude         float32
	Altitude          float64
	Accuracy          float64
	Vertical_accuracy float64
	Bearing           float64
	Speed             float64
	ElapsedMs         uint32
	//Provider string
}

func parseJsonLocation(in []byte) *Location {
	tmp := termuxLocation{}
	err := json.Unmarshal(in, &tmp)
	if err != nil {
		log.Printf("json.Unmarshal of '%v' failed: %v", in, err)
		return nil
	}
	return &Location{
		Lon: tmp.Longitude,
		Lat: tmp.Latitude,
		Alt: tmp.Altitude,
		Acc: tmp.Accuracy,
	}
}

func (p *Provider) RetrieveLocation() *Location {
	p.m.Lock()
	defer p.m.Unlock()
	return p.lastLocation
}

func (p *Provider) updateLocation() {
	ctx, cancel := context.WithTimeout(context.Background(), p.Conf.updateWaitTimeout/2)
	defer cancel()
	cmdline := "termux-location"
	cmd := exec.CommandContext(ctx, cmdline)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("execution of '%s' failed (killed=timeout): %v", cmdline, err)
		return
	}
	now := time.Now()
	newLocation := parseJsonLocation(out)
	if newLocation != nil {
		newLocation.Stamp = now
		p.m.Lock()
		defer p.m.Unlock()
		p.lastLocation = newLocation
	}
}

// continuously retrieve location
// should be run is a goroutine like "go p.Run()"
func (p *Provider) Run(ctx context.Context) {
	ticker := time.NewTicker(p.Conf.UpdateInterval)
	for {
		select {
		case <-ctx.Done():
		case <-ticker.C:
			p.updateLocation()
		}
	}
}
