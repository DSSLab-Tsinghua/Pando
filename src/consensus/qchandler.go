package consensus

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	// "time"

	"pando/src/logging"
	// "pando/src/config"
	"pando/src/cryptolib"
	"pando/src/quorum"
	"pando/src/utils"
	message "pando/src/message"
	pb "pando/src/proto/communication"
	sender "pando/src/communication/sender"
	"pando/src/communication"
	sequence "pando/src/consensus/sequence"
)

/*all the parameter for qc-like protocols*/
var curBlock message.QCBlock       //recently voted block
var curStableBlock message.QCBlock //for QC protocol with f+1 threshold, recently block with 2f+1 QC size
var weakQC message.QCBlock
var votedBlocks utils.IntByteMap //QCP
var lockedBlock message.QCBlock //locked block
var curHash utils.ByteValue     //current hash
var awaitingBlocks utils.IntByteMap	//QCP
var awaitingDecision utils.IntByteMap	//QCP
var awaitingDecisionCopy utils.IntByteMap	//QCP
var statusAfterVC utils.IntValue
var vcAwaitingVotes utils.IntIntMap
var vcBlock message.QCBlock // used in 2-3 protocol
var leaderVCstatus bool     // used in 2-3 protocol
var avgphases int
var weakQCNum int
var normalQCNum int
var strongQCNum int
var par_lockedBlock utils.IntByteMap
var vcTime int64
var lastSeq int
var newView bool    //used in QC protocol to mark new view
var lastViewSeq int //used in QC protocol to mark seq for new view
var vcTracker utils.IntIntMap	//QCP
var vcHashTracker utils.IntByteMap	//QCP
var startTime int64

var qcrlock sync.Mutex
var cblock sync.Mutex
var timerlock sync.Mutex

var validQC message.QCBlock
var lockedQC message.QCBlock
var curlockedBlock message.QCBlock
var commitStatus utils.IntBoolMap
var deliverStatus utils.IntBoolMap
var preHash utils.ByteValue


func GenHashOfTwoVal(input1 []byte, input2 []byte) []byte {
	tmp := make([][]byte, 2, 2)
	tmp[0] = input1
	tmp[1] = input2
	b := bytes.Join(tmp, []byte(""))
	return cryptolib.GenHash(b)
}

//QCP
func ObtainCurHash(input []byte, seq int) []byte {
	var result []byte
	bh := GenHashOfTwoVal(utils.IntToBytes(seq), input)
	ch := curHash.Get()
	result = GenHashOfTwoVal(ch, bh)
	
	return result
}

func CheckContentHash(input []byte, seq int) []byte {
	var result []byte
	bh := GenHashOfTwoVal(utils.IntToBytes(seq), input)
	ch := preHash.Get()
	result = GenHashOfTwoVal(ch, bh)

	return result
}

func FetchBlockInfo(seq int) []byte {
	if seq == 1 || curBlock.Height == 0 {
		return nil //return nil, representing initial block
	}
	var msg []byte
	var err error
	cblock.Lock()
	msg, err = curBlock.Serialize()
	cblock.Unlock()
	if err != nil {
		log.Printf("fail to serialize curblock")
		return []byte("")
	}
	
	return msg
}

func FetchLockedBlockInfo(seq int) []byte {
	// if seq == 1 || curBlock.Height == 0 {
	// 	return nil //return nil, representing initial block
	// }
	var msg []byte
	var err error
	cblock.Lock()
	msg, err = curlockedBlock.Serialize()
	cblock.Unlock()
	if err != nil {
		log.Printf("fail to serialize curlockedBlock")
		return []byte("")
	}

	return msg
}

//QCP
func ProcessQCInfo(hash string, blockinfo message.QCBlock, content message.ReplicaMessage) {
	if blockinfo.PreHash != nil {
		lockedBlock = curBlock
		votedBlocks.Delete(curBlock.Height)
	}

	cblock.Lock()
	curBlock = blockinfo
	cblock.Unlock()
	curHash.Set(curBlock.Hash)
	if !Leader() && blockinfo.Height > curBlock.Height {
		cblock.Lock()
		curBlock = blockinfo
		cblock.Unlock()
		curHash.Set(curBlock.Hash)
	}

	if content.Seq > 3 {
		//awaitingBlocks.Delete(content.Seq-3)
		awaitingDecision.Delete(content.Seq - 3)
		awaitingDecisionCopy.Delete(content.Seq - 3)
	}

	//log.Printf("awaitingDecisionCopy.GetLen() %v %v, %s, %s", content.Seq, awaitingDecisionCopy.GetLen(), awaitingDecisionCopy, curBlock.PrePreHash)
	if curBlock.PrePreHash != nil && sequence.GetSeq() < content.Seq-3 {

		// ch := utils.BytesToString(curBlock.PrePreHash)
		// //shardnum := storage.FetchSeqShard(content.Seq - 3)
		// //shardseqnum := storage.FetchSeqShardSeq(content.Seq - 3)
		// _, any := storage.FetchStoreByKey(ch)
		// if !any { //todo: in production sys, we need to cache the message and deliver it later.
		// 	return
		// }
		// p := fmt.Sprintf("[QC] deliver block at height %d", content.Seq-3)
		// logging.PrintLog(verbose, logging.NormalLog, p)

		/*if vcTime >0 {
			vcdTime := utils.MakeTimestamp()
			log.Printf("deliver block height %v, %v ms", content.Seq-3, vcdTime - vcTime)
		}*/
		//log.Printf("***[%v] deliver request, curSeq %v", content.Seq-3, sequence.GetSeq())
		sequence.UpdateSeq(content.Seq - 3)
		sequence.UpdateLSeq(content.Seq - 3)

	} else {
		sequence.UpdateSeq(content.Seq - 3)
		sequence.UpdateLSeq(content.Seq - 3)
		sequence.UpdateLEpoch(content.Epoch)
	}
	//curStatus.Set(READY)
}

func FetchVCBlock() []byte {
	if vcBlock.View != LocalView() {
		return nil
	}
	msg, _ := vcBlock.Serialize()
	return msg
}

//QCP
func HandleQCByteMsg(inputMsg []byte) {

	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg

	content := message.DeserializeReplicaMessage(input)
	mtype := content.Mtype
	source := content.Source

	/*if !VerifySignature(source, tmp.Version, input, tmp.Sig, content.Seq) {
		p := fmt.Sprintf("[Authentication  Error] Signature of message from Replica %d not verified", source)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}*/

	communication.SetLive(utils.Int64ToString(source))

	//Cache the message if the current view < content view
	/*if LocalView() < content.View {
		if curStatus.Get() != VIEWCHANGE {
			cachedMsgForNextView.Append(inputMsg)
			return
		}
	}*/
	// storage.AddToLog(content.Seq, tmp)

	switch mtype {
	case pb.MessageType_QC:
		HandleDistribute(content)
	case pb.MessageType_QCREP:
		HandleShare(content)
	case pb.MessageType_QC2:
		HandleQCTwoMessage(content)
	case pb.MessageType_QCBC:					//
		HandleQCBCMessage(content)
	case pb.MessageType_QCBCREP:					//
		HandleQCBCRepMessage(content)
	case pb.MessageType_QCBCFINAL:					//
		HandleQCBCFinalMessage(content)
	case pb.MessageType_QCP2:					//
		HandleQCPTwo(content)
	case pb.MessageType_QCP3:					//
		HandleQCPThree(content)
	case pb.MessageType_QCP4:					//
		HandleQCPFour(content)
	case pb.MessageType_NEWVIEW:
		HandleQCPNewView(content)
		
	}
}

/*
return -1 if rank(block) < rank(blocktwo)
		0 if rank(block) = rank(blocktwo)
		1 otherwise
*/
func rank(block message.QCBlock, blocktwo message.QCBlock) int {
	//log.Printf("blockview %v, two view %v, height %v, %v", block.View, blocktwo.View, block.Height, blocktwo.Height)
	if block.View < blocktwo.View {
		return -1
	}
	if block.Height < blocktwo.Height {
		return -1
	} else if block.Height == blocktwo.Height {
		return 0
	}
	return 1
}

func VerifyBlock(curSeq int, blockinfo message.QCBlock) bool {
	if curSeq == 1 {
		return true
	}

	if len(blockinfo.QC) < quorum.KQuorumSize() {
		log.Printf("highQC len is %v but require %v", len(blockinfo.QC), quorum.KQuorumSize())
		return false
	}

	if !VerifyQC(blockinfo) {
		p := fmt.Sprintf("[QC] block signature %d not verified", blockinfo.Height)
		logging.PrintLog(true, logging.ErrorLog, p)
		return false
	}
	return true

}

// //QCP
// func HandleQCMessage(content message.ReplicaMessage) { //For replica to process proposals from the leader @QC
// 	startTime := utils.MakeTimestamp()
// 	//nv := false
// 	hash := ""
// 	source := content.Source

// 	hash = utils.BytesToString(content.Hash)

// 	//if evalMode > 0 && !Leader(){
// 	if evalMode > 0 {
// 		var p = fmt.Sprintf("[Replica] average number of phases per request %v(strongQCNum:%d, normalQCNum:%d, weakQCNum:%d)", avgphases/content.Seq, strongQCNum, normalQCNum, weakQCNum)
// 		logging.PrintLog(true, logging.EvaluationLog, p)
		
// 	}

// 	if vcTime > 0 {
// 		vcdTime := utils.MakeTimestamp()
// 		log.Printf("processing block height %v, %v ms", content.Seq, vcdTime-vcTime)
// 	}
// 	if content.OPS != nil {
// 		awaitingDecision.Insert(content.Seq, content.Hash)
// 		//dTime := utils.MakeTimestamp()
// 		//diff,_ := utils.Int64ToInt(dTime - cTime)
// 		//log.Printf("[%v] ++latency-1 for QCM %v ms", content.Seq, diff)
// 		if !Leader() {
// 			awaitingDecisionCopy.Insert(content.Seq, content.Hash)
// 		}
// 	}
	
// 	log.Printf("handleqc time :%d", int(utils.MakeTimestamp()-startTime))

// }

// func HandleQueue(ch string, ops []pb.RawMessage) {
// 	if Leader() {
// 		return
// 	}
// 	queue.Remove(ch, ops)
	
// 	if queueHead.Get() != "" {
// 		//log.Printf("stop queue head %s", queueHead)
// 		queueHead.Set("")
// 		queueHead.ClearShard()
		
// 		timer.Stop()

// 	}
// }

// //QCP perpare
// func HandleQCRep(content message.ReplicaMessage) {
// 	//startTime := utils.MakeTimestamp()

// 	h, exist := awaitingBlocks.Get(content.Seq)
// 	if exist && bytes.Compare(h, content.Hash) != 0 {
// 		p := fmt.Sprintf("[QC] hash not matching", content.Seq)
// 		logging.PrintLog(true, logging.ErrorLog, p)
// 		return
// 	}
// 	if !cryptolib.VerifySig(content.Source, content.Hash, content.PreHash) {
// 		p := fmt.Sprintf("[QC] signature for QCRep with height %v not verified", content.Seq)
// 		logging.PrintLog(true, logging.ErrorLog, p)
// 		return
// 	}
// 	hash := utils.BytesToString(content.Hash) 

// 	// if config.Consensus() == config.Pando {
// 	// 	commsg := message.ComMessage{
// 	// 		Seq:		content.Epoch,
// 	// 		Process:	"consensus2",
// 	// 		Source:		content.Source,
// 	// 	}
// 	// 	commsgbyte, _ := commsg.Serialize()
	
// 	// 	if !ComProve(content.VrfPi, commsgbyte, commsg.Source, "consensus2") {
// 	// 		p := fmt.Sprintf("[Consensus2] vrfPi from node %v at epoch %v not verified", commsg.Source, content.Epoch)
// 	// 		logging.PrintLog(true, logging.ErrorLog, p)
// 	// 		return 
// 	// 	}
// 	// }

// 	//log.Printf("hanling rep")
// }

func HandleQCTwoMessage(content message.ReplicaMessage) {

	source := content.Source
	//hashstring := utils.BytesToString(content.Hash)
	// if storage.IsDelivered(hashstring) && evalMode == 0 {
	// 	if verbose {
	// 		p := fmt.Sprintf("[QC] No need to process an already delivered Replica message: [%s] seq [%v]", hashstring, content.Seq)
	// 		logging.PrintLog(verbose, logging.NormalLog, p)
	// 	}
	// 	return
	// }

	hash := GenHashOfTwoVal(utils.IntToBytes(0), content.Hash)

	blockinfo := message.DeserializeQCBlock(content.PreHash)
	if !VerifyQC(blockinfo) {
		log.Printf("QC2 for height %v is not verified", blockinfo.Height)
		return
	}
	par_lockedBlock.Insert(content.Seq, content.PreHash)

	msg := message.ReplicaMessage{
		Mtype:    pb.MessageType_QCREP2,
		Source:   id,
		View:     LocalView(),
		Hash:     content.Hash,
		Seq:      content.Seq,
		ShardSeq: content.ShardSeq,
		Shard:    content.Shard,
	}

	sig := cryptolib.GenSig(id, hash)
	msg.PreHash = sig

	msgbyte, err := msg.Serialize()

	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[QCMessage Error] Not able to serialize the message")
		return
	}

	sender.SendToNode(msgbyte, source)
}

// func ConsensusExpireEvent() {
// 	if curStatus.Get() == QCREADY {
// 		curStatus.Set(READY)
// 	} else {
// 		curStatus.Set(QCTimeout)
// 	}
// }
