package utils

import (
	"sync"
	"time"
)

type Snowflake struct {
	timestamp  int64
	datacenter int64
	worker     int64
	sequence   int64
	mutex      sync.Mutex
}

func NewGenerator(datacenter, worker int64) *Snowflake {
	if datacenter > datacenterLimit || worker > workerLimit {
		return nil
	}
	return &Snowflake{
		datacenter: datacenter,
		worker:     worker,
	}
}

const (
	timestampBits   = uint(41)
	datacenterBits  = uint(4)
	workerBits      = uint(13)
	sequenceBits    = uint(5)
	workerShift     = sequenceBits
	datacenterShift = sequenceBits + workerBits
	timestampShift  = sequenceBits + workerBits + datacenterBits
)

const (
	// Starting time period, valid for 69 years from the millisecond timestamp time.
	epoch           = int64(1684307899598)
	timestampLimit  = int64(-1 ^ (-1 << timestampBits))
	datacenterLimit = int64(-1 ^ (-1 << datacenterBits))
	workerLimit     = int64(-1 ^ (-1 << workerBits))
	sequenceLimit   = int64(-1 ^ (-1 << sequenceBits))
)

func (s *Snowflake) NextVal() int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	now := time.Now().UnixNano() / 1000000 // 转毫秒
	// Different IDs are distinguished by incrementing the sequence number
	// within the same millisecond, and waiting for the next millisecond
	// after the sequence bits is full.
	if s.timestamp == now {
		s.sequence = (s.sequence + 1) & sequenceLimit
		if s.sequence == 0 {
			for now <= s.timestamp {
				now = time.Now().UnixNano() / 1000000
			}
		}
	} else {
		s.sequence = 0
	}
	passed := now - epoch
	if passed > timestampLimit {
		return 0
	}
	s.timestamp = passed
	// use or manipulate the resulting ID for stitching
	return (now-epoch)<<timestampShift | (s.datacenter << datacenterShift) | (s.worker << workerShift) | (s.sequence)
}
