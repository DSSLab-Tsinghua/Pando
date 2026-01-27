package consensus

import (
	"log"
	"pando/src/communication/sender"
	"pando/src/config"
	"pando/src/cryptolib"
	"pando/src/utils"
	"pando/src/quorum"
	"sync"
	sequence "pando/src/consensus/sequence"
)

var bcStatus CurStatus
var cachedViewMsg Queue

var numOfNodes int     //n
var curConsensusStatus CurStatus
var curStateTransStatus CurStatus
var curStatusSeq int
var curNewViewStatus CurStatus

var started bool        // whether the protocol has been started

type ConsensusType int

const (
	Pando           ConsensusType = 4
)

type RbcType int

const (
	RBC   RbcType = 0
	ECRBC RbcType = 1
)

type TypeOfBuffer int32

const (
	BUFFER TypeOfBuffer = 0
	CACHE  TypeOfBuffer = 1
)

type ConsensusStatus int

const (
	PREPREPARED ConsensusStatus = 0
	PREPARED    ConsensusStatus = 1
	COMMITTED   ConsensusStatus = 2
	BOTHPANDC   ConsensusStatus = 3
)

func GetInstanceID(input int) int {
	return input + n*epoch.Get() //baseinstance*epoch.Get()
}

func GetIndexFromInstanceID(input int, e int) int {
	return input - n*e
}

func GetInstanceIDsOfEpoch() []int {
	var output []int
	for i := 0; i < len(members); i++ {
		output = append(output, GetInstanceID(members[i]))
	}
	return output
}

func StartHandler(rid string, mem string) {
	id, errs = utils.StringToInt64(rid)
	if errs != nil {
		log.Printf("[Error] Replica id %v is not valid. Double check the configuration file", id)
		return
	}
	iid, _ = utils.StringToInt(rid)

	config.LoadConfig()
	cryptolib.StartCrypto(id, config.CryptoOption())
	consensus = ConsensusType(config.Consensus())

	n = config.FetchNumReplicas()
	curStatus.Init()
	epoch.Init()
	midTime = make(map[int]int64)
	queue.Init()
	MsgQueue.Init()
	sender.StartSender(rid)
	StartVChandler()

	verbose = config.FetchVerbose()
	sleepTimerValue = config.FetchSleepTimer()
	log.Printf("sleeptimer value %v", sleepTimerValue)

	switch consensus {
	case Pando:
		InitializeMembership()
		numOfNodes = config.FetchNumReplicas()
		quorum.StartPandoQuorum(numOfNodes, 0)
		evalInterval = config.EvalInterval()
		curOPS.Init()
		totalOPS.Init()

		SetView(view)
		cryptolib.StartECDSA(id)

		log.Printf(">>>Running Pando protocol!!!")

		// qchandler
		votedBlocks.Init()
		awaitingBlocks.Init()
		awaitingDecision.Init()
		awaitingDecisionCopy.Init()
		vcTracker.Init()
		vcHashTracker.Init()
		commitStatus.Init()
		deliverStatus.Init()

		newView = false
		normalQCNum = 0
		weakQCNum = 0
		avgphases = 0

		//qcp
		globalStatus.Init()

		//pando
		consensusGlobalStatus.Init()
		epochMap.Init()
		epochCount.Init()
		epochStatus.Init()

		//sequence.EpochLIncrement()
		InitPando()

		sequence.ResetSeq(0)
		sequence.ResetLSeq(0)

		// JoinSystem(0,0,0,0) //0,0,0,0
		// go MonitorJoinMessage()

		log.Printf("Start Pando transmission process...")
		go BroadcastMonitor()

		log.Printf("Start Pando consensus process...")
		go NewViewMonitor()
		go RequestMonitor() //epoch mod n == id

		// sequence.ResetSeq(1)
		// go RetrievalMonitor()

		//Run TimerMonitor routine for all node, but in leader node the routine do nothing
		//go TimerMonitor()

		//go CachedMsgMonitor()

	default:
		log.Fatalf("Consensus type not supported")
	}

	started = true

}

type CurStatus struct {
	enum Status
	sync.RWMutex
}

type Status int

const (
	READY      Status = 0
	PROCESSING Status = 1
	VIEWCHANGE Status = 2
	SLEEPING   Status = 3
	RECOVERING Status = 4
	BLOCK      Status = 6
)

func (c *CurStatus) Set(status Status) {
	c.Lock()
	defer c.Unlock()
	c.enum = status
}

func (c *CurStatus) Init() {
	c.Lock()
	defer c.Unlock()
	c.enum = READY
}

func (c *CurStatus) Get() Status {
	c.RLock()
	defer c.RUnlock()
	return c.enum
}

func (c *CurStatus) InitWithBlock() {
	c.Lock()
	defer c.Unlock()
	c.enum = BLOCK
}
