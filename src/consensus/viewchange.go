package consensus

import (
	// "fmt"
	// "log"
	// "pando/src/communication/sender"
	"pando/src/config"
	// "pando/src/cryptolib"
	"pando/src/logging"
	// "pando/src/message"
	// pb "pando/src/proto/communication"
	"pando/src/quorum"
	"pando/src/utils"
	"sync"
	"time"
	// sequence "pando/src/consensus/sequence"
)

var view int
var viewMux sync.RWMutex
var leader bool
var endorser bool
var PandoMembers utils.IntArr  
var curMaxID int

// var rotatingTimer *time.Timer  // wcx: no use.
var timeoutBuffer quorum.INTBUFFER

const UintSize = 32 << (^uint(0) >> 32 & 1)

//Initialize members with default value
func InitializeDefaultMembers(n int) {
	for i := 0; i < n; i++ {
		PandoMembers.Add(i)
	}
}

//Initialize members from config file
func InitializeMembership() {
	nodes := config.FetchNodes()
	for i := 0; i < len(nodes); i++ {
		id, _ := utils.StringToInt(nodes[i])
		PandoMembers.Add(id)
	}
	PandoMembers.Sorts()
	// memberscopy.Set(PandoMembers.GetAll())
}

func InitializeDynamicMembership() {
	mem := PandoMembers.GetAll()
	curMaxID = mem[len(mem)-1]
	// memberscopy.Set(PandoMembers.GetAll())
}


// ID of the leader according to the view number
func LeaderID(view int) int {
	return LeaderDyMem(view)
}

func CheckLeaderID(epoch int) int64 {
	return int64(LeaderPandoMem(epoch))
}

func CheckLeader(epoch int, jid int64) bool {	//change leader for each epoch
	tmp, _ := utils.Int64ToInt(jid)

	if LeaderPandoMem(epoch) == tmp {
		return true
	} else {
		return false
	}
}

//return the ID of the next leader under dynamic membership
//QCP
func LeaderDyMem(view int) int {
	mem := PandoMembers.GetAll()//mem is [0,1,2,3]

	if config.FetchNumReplicas() == 0 {
		time.Sleep(time.Duration(5) * time.Millisecond)
	}
	if config.FetchNumReplicas() == 0 {
		logging.PrintLog(true, logging.ErrorLog, "Failed to get a correct leader according to the view number")
		return mem[0]
	}
	return mem[view%config.FetchNumReplicas()]
}

//QCP
func LeaderPandoMem(epoch int) int {
	// if curStatus.Get() == VIEWCHANGE {
	// 	mem := memberscopy.GetAll()
	// 	return mem[epoch%len(mem)]
	// }
	mem := PandoMembers.GetAll()//mem is [0,1,2,3,...]

	if config.FetchNumReplicas() == 0 {
		time.Sleep(time.Duration(5) * time.Millisecond)
	}
	if config.FetchNumReplicas() == 0 {
		logging.PrintLog(true, logging.ErrorLog, "Failed to get a correct leader according to the epoch number")
		return mem[0]
	}
	return mem[epoch%config.FetchNumReplicas()]
}

// func Endorser() bool {
// 	return endorser
// }

// Check whether this node is the leader
func Leader() bool {
	return leader
}

// Set up leader status, returns true if this node is the leader
func SetLeader(input bool) {
	leader = input
}

// Return view number
func LocalView() int {
	return view
}

func InitView() {
	view = 0
}

// Set view number
func SetView(v int) {
	view = v
	viewInt := utils.IntValue{}
	viewInt.Set(view)

	tmp, _ := utils.Int64ToInt(id)
	if LeaderID(v) == tmp {
		leader = true
	} else {
		leader = false
	}
}

/*
By default, view starts from 0.
//QCP
*/
func StartVChandler() {
	view = 0
}
