package local

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3" // init the sqlite driver
	"log"
	"os"
	"github.com/tinygoprogs/sigint/wifi"
	"strings"
)

const ChanSizeDefault = 0x100
const NmapsDefault = 0x10
const FileDefault = "sigint-wifi-devices.db"

// can be passed empty, sane defaults will be choosen
type LocalConfig struct {
	// sqlite file name
	File string
	// a human can have this manny devices associated
	Nmaps int
	// internal channel sizes
	ChanSize int
}

/* Implements wifi.CollectorServer, but locally with sqlite.

Note: Could easily be expanded by registering an instance with grpc, so we can
be a remote Collector. But should also move to a more serious DB, e.g.
PostgreSQL. */
type LStore struct {
	db         *sql.DB
	push       chan *wifi.Device
	flush_done chan bool
}

/* Create a new local storage.

Note: Should Wait() on the LStore before exiting the program, to ensure all
data is persisted. */
func NewLStore(ctx context.Context, cnf *LocalConfig) (ls *LStore, err error) {
	if cnf.ChanSize == 0 {
		cnf.ChanSize = ChanSizeDefault
	}
	if cnf.File == "" {
		cnf.File = FileDefault
	}
	if cnf.Nmaps == 0 {
		cnf.Nmaps = NmapsDefault
	}
	ls = &LStore{
		push:       make(chan *wifi.Device, cnf.ChanSize),
		flush_done: make(chan bool, 1),
	}
	execute_init := false
	if _, err := os.Stat(cnf.File); os.IsNotExist(err) {
		execute_init = true
	}
	err = ls.open(cnf.File)
	if err != nil {
		return
	}
	if execute_init {
		for _, stmt := range createStmts(cnf.Nmaps) {
			_, err = ls.db.Exec(stmt)
			if err != nil {
				return
			}
		}
	}
	go ls.sql_io(ctx)
	return
}

// persist new devices
func (ls *LStore) NewDevices(ctx context.Context, devs *wifi.Devices) (*wifi.Ack, error) {
	var ndevs, ndps int
	for _, dev := range devs.GetDevices() {
		ls.push <- dev
		ndps += len(dev.GetDataPoints())
		ndevs++
	}
	return &wifi.Ack{
		NDataPoints: int32(ndps),
		NDevices:    int32(ndevs),
	}, nil
}

// persist new mapping
func (ls *LStore) NewMapping(ctx context.Context, devs *wifi.HumanMapping) (*wifi.Ack, error) {
	return nil, errors.New("not implemented")
}

// wait for data to be persisted
func (ls *LStore) Wait() {
	<-ls.flush_done
}

func (ls *LStore) sql_io(ctx context.Context) {
	var err error
	for {
		select {
		case dev := <-ls.push:
			log.Printf("store(%v)", dev.MAC)
			err = ls.store(dev)
			if err != nil {
				log.Printf("failed to store: %v, %v, data is lost now!", dev, err)
			}
		case <-ctx.Done():
			ls.shutdown()
			return
		}
	}
}

func (ls *LStore) shutdown() {
endfor:
	for {
		select {
		case dev := <-ls.push:
			err := ls.store(dev)
			if err != nil {
				log.Printf("failed to store: %v, %v, data is lost now!", dev, err)
			}
		default:
			break endfor
		}
	}
	log.Print("closing db")
	ls.db.Close()
	ls.flush_done <- true
}

func (ls *LStore) open(file string) (err error) {
	ls.db, err = sql.Open("sqlite3", file)
	if err == nil {
		_, err = ls.db.Exec("PRAGMA foreign_keys = true;")
	}
	return
}

func (ls *LStore) store(dev *wifi.Device) (err error) {
	var (
		res     driver.Result
		row     *sql.Row
		node_id int64
	)
	if dev == nil || dev.MAC == "" {
		return errors.New("dude filter your shit")
	}

	// insert a new node, or get the node id via Query()
	row = ls.db.QueryRow("SELECT id FROM nodes WHERE addr = ?", dev.MAC)
	err = row.Scan(&node_id)
	if err != nil {
		res, err = ls.db.Exec("INSERT INTO nodes(addr) VALUES(?)", dev.MAC)
		if err != nil {
			return
		}
		node_id, err = res.LastInsertId()
		if err != nil {
			return
		}
	}

	if node_id == 0 {
		return errors.New("no node_id found!")
	}

	// insert device info
	for _, dp := range dev.GetDataPoints() {
		var err error
		res, err = ls.db.Exec(`INSERT -- maybe.. OR IGNORE
      INTO datapoints(time, frequency, signal, longitude, latitude, node_id)
      VALUES(?, ?, ?, ?, ?, ?)`,
			dp.TimeStamp, dp.Frequency, dp.Signal, dp.Location.Lon, dp.Location.Lat, node_id)
		if err != nil {
			log.Printf("insert datapoint failed: %v", err)
		}
	}
	return
}

// return an array of table creation statements with max <n> devices per human
// this is an sqlite limitation (no array of foreign keys)
func createStmts(n int) []string {
	var (
		ids         []string
		keys        []string
		id_base     string = "node_id%d INTEGER"
		key_base    string = "FOREIGN KEY(node_id%d) REFERENCES nodes(id)"
		sep         string = ",\n    "
		hd_map_base string = "\n    name STRING"
		hd_map      string
	)
	for i := 0; i < n; i++ {
		ids = append(ids, fmt.Sprintf(id_base, i))
		keys = append(keys, fmt.Sprintf(key_base, i))
	}
	hd_map = hd_map_base +
		sep + strings.Join(ids, sep) +
		sep + strings.Join(keys, sep)

	creat := "CREATE TABLE %s (id INTEGER PRIMARY KEY, %s)"

	return []string{
		fmt.Sprintf(creat, "nodes", `
      addr STRING UNIQUE
    `),
		fmt.Sprintf(creat, "humans", hd_map),
		fmt.Sprintf(creat, "datapoints", `
      time BLOB,
      frequency INTEGER,
      signal INTEGER,
      longitude INTEGER,
      latitude INTEGER,
      node_id INTEGER,
      FOREIGN KEY(node_id) REFERENCES nodes(id) ON DELETE CASCADE,
      -- a single device should only be able to send one frame at a time
      CONSTRAINT unique_dps UNIQUE (time, node_id, signal)
    `),
	}
}
