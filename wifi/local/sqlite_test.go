package local

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"github.com/tinygoprogs/sigint/wifi"
	"testing"
	"time"
)

var dbname = flag.String("testdb", "test.db", "test database path")

func init() {
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	if !testing.Verbose() {
		log.SetOutput(ioutil.Discard)
	}
}

func TestLStoreStoreDeviceAndQueryIt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ls, err := NewLStore(ctx, &LocalConfig{File: *dbname, Nmaps: 10, ChanSize: 2})
	if err != nil {
		t.Errorf("db creation failed: %v", err)
	}

	stmp := uint64(time.Now().UnixNano())
	dev := &wifi.Device{
		MAC: "11:22:33:44:55:66",
		DataPoints: []*wifi.DataPoint{
			&wifi.DataPoint{
				Signal: uint32(1), Frequency: uint32(2),
				TimeStamp: stmp, Location: &wifi.Coordinates{},
			},
		},
	}

	ls.push <- dev
	cancel()
	ls.Wait()
	// TODO: check if device is persisted
	//os.Remove(*dbname)
}
