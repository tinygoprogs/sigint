package wifi

import (
	"fmt"
	"log"
	"sync"
)

type PacketStats struct {
	stats map[string]int
	mtx   sync.RWMutex
}

func (s *PacketStats) inc(which string) {
	s.mtx.Lock()
	s.stats[which]++
	s.mtx.Unlock()
}

func (s *PacketStats) log() {
	s.mtx.RLock()
	log.Print(s.String())
	s.mtx.RUnlock()
}

func (s *PacketStats) String() string {
	return fmt.Sprintf("PacketStats[%v]", s.stats)
}

func NewPacketStats(values ...string) *PacketStats {
	s := PacketStats{stats: make(map[string]int, len(values))}
	for _, val := range values {
		s.stats[val] = 0
	}
	return &s
}
