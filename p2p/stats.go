package p2p

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

type BlockInsertCounter struct {
	Blocks_Accepted uint64
	Blocks_Rejected uint64
}

type BlockSendingError struct {
	Block_Type          string
	When                time.Time
	Error_Message       string
	Destination_Peer_ID uint64
}

type BlockReceivingError struct {
	Block_Type    string
	When          time.Time
	Error_Message string
	From_Peer_ID  uint64
}

type BlockCollisionError struct {
	Block_Type    string
	When          time.Time
	Error_Message string
	Peer_ID       uint64
	Incoming      bool
}

type PeerStats struct {
	Sending_Errors   []BlockSendingError
	Receiving_Errors []BlockReceivingError
	Collision_Errors []BlockCollisionError
}

var Stats_mutex sync.Mutex
var Pstat = make(map[string]PeerStats)
var BlockInsertCount = make(map[string]BlockInsertCounter)

func LogAccept(Address string) {

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	Address = ParseIPNoError(Address)

	stat := BlockInsertCount[Address]
	stat.Blocks_Accepted++
	BlockInsertCount[Address] = stat

}

func LogReject(Address string) {

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	Address = ParseIPNoError(Address)

	stat := BlockInsertCount[Address]
	stat.Blocks_Rejected++
	BlockInsertCount[Address] = stat

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

	for Address, _ := range Pstat {
		var new_peer_stat PeerStats
		Pstat[Address] = new_peer_stat
	}

	for Address := range BlockInsertCount {
		stat := BlockInsertCount[Address]
		stat.Blocks_Accepted = 0
		stat.Blocks_Rejected = 0
		BlockInsertCount[Address] = stat
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

	_, x := Pstat[Address]
	if x {
		var new_peer_stat PeerStats
		Pstat[Address] = new_peer_stat
	}

	stat, y := BlockInsertCount[Address]
	if y {
		stat.Blocks_Accepted = 0
		stat.Blocks_Rejected = 0
		BlockInsertCount[Address] = stat
	}
}

func PeerLogConnectionFail(Address string, Block_Type string, PeerID uint64, Message string) {

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	Address = ParseIPNoError(Address)

	peer := Pstat[Address]

	is_collision := regexp.MustCompile("^collision ")
	is_tip_issue := regexp.MustCompile("^tip could not be expanded")

	if is_collision.Match([]byte(Message)) || is_tip_issue.Match([]byte(Message)) {

		stat := peer.Collision_Errors

		var Error BlockCollisionError
		Error.Block_Type = Block_Type
		Error.When = time.Now()
		Error.Error_Message = Message
		Error.Incoming = false
		Error.Peer_ID = PeerID

		stat = append(stat, Error)

		peer.Collision_Errors = stat

	} else {
		// Log error

		stat := peer.Sending_Errors

		var Error BlockSendingError
		Error.Block_Type = Block_Type
		Error.When = time.Now()
		Error.Error_Message = Message
		Error.Destination_Peer_ID = PeerID

		stat = append(stat, Error)

		peer.Sending_Errors = stat
	}

	Pstat[Address] = peer
	go Peer_SetFail(Address)
	go logger.V(2).Info(fmt.Sprintf("Error (%s) - Logged for Connection: %s", Message, Address))

}

func PeerLogReceiveFail(Address string, Block_Type string, PeerID uint64, Message string) {

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	Address = ParseIPNoError(Address)

	peer := Pstat[Address]

	is_collision := regexp.MustCompile("^collision ")
	is_tip_issue := regexp.MustCompile("^tip could not be expanded")

	if is_collision.Match([]byte(Message)) || is_tip_issue.Match([]byte(Message)) {

		stat := peer.Collision_Errors

		var Error BlockCollisionError
		Error.Block_Type = Block_Type
		Error.When = time.Now()
		Error.Error_Message = Message
		Error.Incoming = true
		Error.Peer_ID = PeerID

		stat = append(stat, Error)

		peer.Collision_Errors = stat

	} else {
		// Log error
		stat := peer.Receiving_Errors

		var Error BlockReceivingError
		Error.Block_Type = Block_Type
		Error.When = time.Now()
		Error.Error_Message = Message
		Error.From_Peer_ID = PeerID

		stat = append(stat, Error)

		peer.Receiving_Errors = stat
	}
	Pstat[Address] = peer

	go Peer_SetFail(Address)
	go logger.V(2).Info(fmt.Sprintf("Error (%s) - Logged for Connection: %s", Message, Address))

}

func GetPeerBTS(Address string) (Accepted uint64, Rejected uint64, Total uint64, SuccessRate float64) {

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	Address = ParseIPNoError(Address)

	stat, ps := BlockInsertCount[Address]
	if ps {

		total := float64(stat.Blocks_Accepted + stat.Blocks_Rejected)
		SuccessRate = (float64(stat.Blocks_Accepted) / total) * 100

		return stat.Blocks_Accepted, stat.Blocks_Rejected, (stat.Blocks_Accepted + stat.Blocks_Rejected), SuccessRate
	}

	return Accepted, Rejected, Total, SuccessRate
}
