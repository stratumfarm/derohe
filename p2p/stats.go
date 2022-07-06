package p2p

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/deroproject/derohe/block"
	"github.com/deroproject/derohe/config"
	"github.com/deroproject/derohe/globals"
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

type MyBlockReceivingError struct {
	Block_Type    string
	When          time.Time
	Error_Message string
}

type MiniBlockLog struct {
	Miniblock   block.MiniBlock
	NodeAddress string
}

var MiniblockLogs = make(map[string]MiniBlockLog)

type FinalBlockLog struct {
	Block       block.Block
	NodeAddress string
}

var FinalBlockLogs = make(map[string]FinalBlockLog)

var Stats_mutex sync.Mutex

var Pstat = make(map[string]PeerStats)
var BlockInsertCount = make(map[string]BlockInsertCounter)

var Selfish_mutex sync.Mutex
var SelfishNodeStats = make(map[string][]MyBlockReceivingError)

var log_miniblock_mutex sync.Mutex

func GetNodeFromMiniHash(hash string) (Address string) {

	log_miniblock_mutex.Lock()
	defer log_miniblock_mutex.Unlock()

	for block_hash, block := range MiniblockLogs {

		if hash == block_hash {
			return block.NodeAddress
		}
	}

	for block_hash, block := range FinalBlockLogs {

		if hash == block_hash {
			return block.NodeAddress
		}
	}

	return Address
}

func GetFinalBlocksFromHeight(height uint64) map[string]block.Block {

	log_miniblock_mutex.Lock()
	defer log_miniblock_mutex.Unlock()

	var Blocks = make(map[string]block.Block)

	for hash, block := range FinalBlockLogs {
		if block.Block.Height+100 >= height {
			Blocks[hash] = block.Block
		} else {
			delete(FinalBlockLogs, hash)
		}
	}

	return Blocks

}

func GetMiniBlocksFromHeight(height uint64) map[string]block.MiniBlock {

	log_miniblock_mutex.Lock()
	defer log_miniblock_mutex.Unlock()

	var Blocks = make(map[string]block.MiniBlock)

	for hash, block := range MiniblockLogs {
		if block.Miniblock.Height+99 > height {
			Blocks[hash] = block.Miniblock
		} else {
			delete(MiniblockLogs, hash)
		}
	}

	return Blocks

}

func GetMiniBlockNodeStats(height uint64) map[string]int {

	log_miniblock_mutex.Lock()
	defer log_miniblock_mutex.Unlock()

	var BlocksPerAddress = make(map[string]int)

	for _, block := range MiniblockLogs {
		if block.Miniblock.Height+99 > height {
			BlocksPerAddress[block.NodeAddress]++
		}
	}

	return BlocksPerAddress
}

func LogFinalBlock(bl block.Block, Address string) {

	log_miniblock_mutex.Lock()
	defer log_miniblock_mutex.Unlock()

	BlockHash := fmt.Sprintf("%s", bl.GetHash())
	Address = ParseIPNoError(Address)

	stat, found := FinalBlockLogs[BlockHash]

	if !found {
		stat.Block = bl
		stat.NodeAddress = Address
	} else {
		if bl.Timestamp < stat.Block.Timestamp {
			stat.NodeAddress = Address
		}
	}

	FinalBlockLogs[BlockHash] = stat
}

func LogMiniblock(mbl block.MiniBlock, Address string) {

	log_miniblock_mutex.Lock()
	defer log_miniblock_mutex.Unlock()

	MiniblockHash := fmt.Sprintf("%s", mbl.GetHash())
	Address = ParseIPNoError(Address)

	stat, found := MiniblockLogs[MiniblockHash]

	if !found {
		stat.Miniblock = mbl
		stat.NodeAddress = Address
	} else {
		if mbl.Timestamp < stat.Miniblock.Timestamp {
			stat.NodeAddress = Address
		}
	}

	MiniblockLogs[MiniblockHash] = stat
}

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

	go ClearPstat()

	peer_mutex.Lock()
	defer peer_mutex.Unlock()

	for _, p := range peer_map {
		p.FailCount = 0
		p.GoodCount = 0
	}

}

func ClearPstat() {
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

func PstatCount() (total_peer_sending_error_count int, total_peer_receiving_error_count int, collision_count int) {

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	for _, ps := range Pstat {
		total_peer_sending_error_count += len(ps.Sending_Errors)
		total_peer_receiving_error_count += len(ps.Receiving_Errors)
		collision_count += len(ps.Collision_Errors)
	}

	return total_peer_sending_error_count, total_peer_receiving_error_count, collision_count
}

func ClearPeerStats(Address string) {

	Address = ParseIPNoError(Address)

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

func SelfishNodeCounter(Address string, Block_Type string, PeerID uint64, Message string, BlockData []byte) {

	Selfish_mutex.Lock()
	defer Selfish_mutex.Unlock()

	Address = ParseIPNoError(Address)

	// If errors showing connection error, then log this so peer can get cleaned up
	connection_down := regexp.MustCompile("^connection is shut down")
	closed_pipe := regexp.MustCompile("io: read/write on closed pipe")

	if !connection_down.Match([]byte(Message)) && !closed_pipe.Match([]byte(Message)) {

		// Check if collision and if it's valid
		//fmt.Errorf("collision %x", mbl.Serialize()), false
		is_collision := regexp.MustCompile("^collision ")
		if is_collision.Match([]byte(Message)) {

			res := strings.TrimPrefix(Message, "collision ")

			if res == fmt.Sprintf("%x", BlockData) {
				logger.Info(fmt.Sprintf("Node (%s) replied with collision and it appears to be genuine", Address))
			} else {
				logger.Info(fmt.Sprintf("Selfish Node (%s) identified - replied with BAD collision message (%s) vs (%x)", Address, res, BlockData))
			}
		}
		var Error MyBlockReceivingError

		Error.Block_Type = Block_Type
		Error.When = time.Now()
		Error.Error_Message = Message

		logs := SelfishNodeStats[Address]
		logs = append(logs, Error)
		SelfishNodeStats[Address] = logs

	}

}

func GetPeerRBS(Address string) (Collisions uint64, CollisionRate float64, TIPFailCount uint64, TIPFailRate float64) {

	Address = ParseIPNoError(Address)

	Selfish_mutex.Lock()
	defer Selfish_mutex.Unlock()

	is_tip_issue := regexp.MustCompile("^tip could not be expanded")
	is_collision := regexp.MustCompile("^collision ")

	logs, x := SelfishNodeStats[Address]

	Collisions = 0
	TIPFailCount = 0

	if x {
		for _, log := range logs {

			if is_collision.Match([]byte(log.Error_Message)) {
				Collisions++
			}
			if is_tip_issue.Match([]byte(log.Error_Message)) {
				TIPFailCount++
			}
		}
	}

	if globals.BlocksMined < 1 {
		return Collisions, float64(0), TIPFailCount, float64(0)
	}

	CollisionRate = 0
	TIPFailRate = 0

	if Collisions >= 1 {
		CollisionRate = float64((float64(Collisions) / float64(globals.BlocksMined)) * 100)
	}

	if TIPFailCount >= 1 {
		TIPFailRate = float64((float64(TIPFailCount) / float64(globals.BlocksMined)) * 100)
	}

	return Collisions, CollisionRate, TIPFailCount, TIPFailRate
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

		// check collision is genuine

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

	// If errors showing connection error, then log this so peer can get cleaned up
	connection_down := regexp.MustCompile("^connection is shut down")
	closed_pipe := regexp.MustCompile("io: read/write on closed pipe")

	if connection_down.Match([]byte(Message)) || closed_pipe.Match([]byte(Message)) {
		go Peer_SetFail(Address)
	}

	Pstat[Address] = peer

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

func ClearPeerLogsCron() {

	Stats_mutex.Lock()
	defer Stats_mutex.Unlock()

	cleared_counter := 0
	for peer, stat := range Pstat {

		var Sending_Errors []BlockSendingError
		var Receiving_Errors []BlockReceivingError
		var Collision_Errors []BlockCollisionError

		for _, log := range stat.Sending_Errors {
			if log.When.Unix()+config.RunningConfig.ErrorLogExpirySeconds > time.Now().Unix() {
				Sending_Errors = append(Sending_Errors, log)
			} else {
				cleared_counter++
			}
		}

		for _, log := range stat.Receiving_Errors {
			if log.When.Unix()+config.RunningConfig.ErrorLogExpirySeconds > time.Now().Unix() {
				Receiving_Errors = append(Receiving_Errors, log)
			} else {
				cleared_counter++
			}
		}

		for _, log := range stat.Collision_Errors {
			if log.When.Unix()+config.RunningConfig.ErrorLogExpirySeconds > time.Now().Unix() {
				Collision_Errors = append(Collision_Errors, log)
			} else {
				cleared_counter++
			}
		}

		stat.Sending_Errors = Sending_Errors
		stat.Receiving_Errors = Receiving_Errors
		stat.Collision_Errors = Collision_Errors

		if len(stat.Sending_Errors) == 0 && len(stat.Receiving_Errors) == 0 && len(stat.Collision_Errors) == 0 {
			delete(Pstat, peer)
		} else {
			Pstat[peer] = stat
		}
	}

	logger.V(2).Info(fmt.Sprintf("Cleared (%d) peer logs", cleared_counter))
}
