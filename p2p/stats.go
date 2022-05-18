package p2p

import (
	"sync"
	"time"
)

type BlockInsertCounter struct {
	Blocks_Accepted uint64
	Blocks_Rejected uint64
}

type BlockSendingError struct {
	Block_ID            string
	Block_Type          string
	When                time.Time
	Error_Message       string
	Destination_Peer_ID uint64
	Address             string
}

type BlockReceivingError struct {
	Block_ID      string
	Block_Type    string
	When          time.Time
	Error_Message string
	From_Peer_ID  uint64
	Address       string
}

type BlockCollisionError struct {
	Block_ID      string
	Block_Type    string
	When          time.Time
	Error_Message string
	Peer_ID       uint64
	Address       string
	Incoming      bool
}

type PeerStats struct {
	Peer_ID          uint64
	Address          string
	Sending_Errors   []*BlockSendingError
	Receiving_Errors []*BlockReceivingError
	Collision_Errors []*BlockCollisionError
}

var Pstat = map[string]*PeerStats{}

var Stats_mutex sync.Mutex

var BlockInsertCount = map[string]*BlockInsertCounter{}

func LogAccept(Address string) {

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	Address = ParseIPNoError(Address)

	_, stat := BlockInsertCount[Address]
	if !stat {
		var NewMetric BlockInsertCounter
		NewMetric.Blocks_Accepted++
		BlockInsertCount[Address] = &NewMetric
	} else {
		BlockInsertCount[Address].Blocks_Accepted++
	}

}

func LogReject(Address string) {

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	Address = ParseIPNoError(Address)

	_, stat := BlockInsertCount[Address]
	if !stat {
		var NewMetric BlockInsertCounter
		NewMetric.Blocks_Rejected++
		BlockInsertCount[Address] = &NewMetric
	} else {
		BlockInsertCount[Address].Blocks_Rejected++

	}

}

func ClearAllStats() {

	peer_mutex.Lock()
	defer peer_mutex.Unlock()

	for _, p := range peer_map {
		p.FailCount = 0
		p.GoodCount = 0
	}

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	var newPstat = map[string]*PeerStats{}
	Pstat = newPstat

	for _, counter := range BlockInsertCount {
		counter.Blocks_Accepted = 0
		counter.Blocks_Rejected = 0
	}
}

func ClearPeerStats(Address string) {

	peer_mutex.Lock()
	defer peer_mutex.Unlock()

	Address = ParseIPNoError(Address)

	for _, p := range peer_map {
		if Address == ParseIPNoError(p.Address) {
			p.FailCount = 0
			p.GoodCount = 0
		}
	}

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	for _, p := range Pstat {
		if Address == p.Address {
			var peer_stats PeerStats
			peer_stats.Address = Address
			Pstat[Address] = &peer_stats
		}
	}

	for key, p := range BlockInsertCount {
		if Address == key {
			p.Blocks_Accepted = 0
			p.Blocks_Rejected = 0
		}
	}
}
