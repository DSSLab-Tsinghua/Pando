package consensus

import (
	// "bytes"
	"fmt"
	"log"
	"sync"
	"time"

	"pando/src/logging"
	"pando/src/config"
	"pando/src/cryptolib"
	"pando/src/quorum"
	"pando/src/utils"
	message "pando/src/message"
	pb "pando/src/proto/communication"
	sender "pando/src/communication/sender"
	// "pando/src/communication"
	sequence "pando/src/consensus/sequence"
	bucket "pando/src/consensus/bucket"
)

var globalStatus utils.IntBoolMap //QCP
var qclock sync.Mutex
var qc1lock sync.Mutex
var qc2lock sync.Mutex
var qc3lock sync.Mutex

var epochMap utils.IntBytesMap   //QCP
var epochCount utils.IntIntMap   //QCP
var epochStatus utils.IntBoolMap //QCP
var lastVCEpoch utils.IntValue
var curTransmissionQC utils.IntByteMap
var epochOPSMap utils.IntBytesMap
var TransmissionStatus utils.IntBoolMap
var NVcounter utils.IntIntMap

var ProtocolMode = 1 // two protocol modes
var Pole2 = false
var Pole2Freq = 20

// QCP
func BroadcastMonitor() {
	//looping execution
	for {

		if !(bcStatus.Get() == READY) || queue.IsEmpty() {
			//log.Printf("waiting for client request...")
			time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
			continue
		}

		tmpseq := sequence.GetEpoch()
		if tmpseq > 0 {
			transStatus, _ := TransmissionStatus.Get(tmpseq)
			status, _ := globalStatus.Get(tmpseq)
			if !(transStatus && status) || !(bcStatus.Get() == READY) || queue.IsEmpty() {
				time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
				continue
			}
			//trans_t2.Insert(tmpseq, utils.MakeTimestamp())
		}

		bcStatus.Set(PROCESSING)
		seq := sequence.EpochIncrement() //Epoch++

		if seq >= 2 {
			pando_t1.Insert(seq, utils.MakeTimestamp())
			log.Printf("catch pando_t1 for seq %v...", seq)
		}

		var batch []pb.RawMessage
		batch = queue.GrabWtihMaxLenAndClear()
		//batch = queue.GrabWithMaxLen()	//queue队列里的msg不删除
		batchSize.Insert(seq, len(batch))

		msg := message.ReplicaMessage{
			Mtype:  pb.MessageType_QCBC,
			Seq:    seq,
			Source: id,
			View:   len(batch),
			OPS:    batch,
			TS:     utils.MakeTimestamp(),
			Num:    quorum.NSize(),
		}
		tmphash := msg.GetMsgHash()
		msg.Hash = ObtainCurHash(tmphash, seq)

		if seq > 1 {
			qcblockbytes, _ := curTransmissionQC.Get(seq - 1)
			msg.PreHash = qcblockbytes
		} else {
			msg.PreHash = []byte("")
		}

		// evalTest, _ := epochMap.Get(seq)
		// log.Printf("[Evaluation test] EvalTest in seq %v is %v.", seq, evalTest)

		log.Printf("node %v broadcasting... %v", id, seq)
		msgbyte, err := msg.Serialize()
		//log.Printf("proposing block with height %v", seq)

		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[Replica Error] Pando Not able to serialize the QCBC message.")
		} else {
			sender.RBCByteBroadcast(msgbyte)
		}
		request, _ := message.SerializeWithSignature(id, msgbyte)
		HandleQCByteMsg(request)

	}
}

func HandleQCBCMessage(content message.ReplicaMessage) { //todo: verify hash
	//Verify QC and vrf
	qcblockinfo := message.DeserializeQCBlock(content.PreHash)
	if !VerifyQC(qcblockinfo) {
		log.Printf("Failed to verifyQC in seq %v, qcblock seq is %v.", content.Seq, qcblockinfo.Height)
		return
	}

	if !VerifyVrf(qcblockinfo, "transmission") {
		log.Printf("Failed to VRF in seq %v, transmission qcblock seq is %v.", content.Seq, qcblockinfo.Height)
		return
	}

	var tempres bool
	totalReplicas := config.FetchNumReplicas()
	index := (totalReplicas * (content.Seq - 1)) + int(id)
	tempres, _ = cTransIsCommitteeMap.Get(index)

	VrfPi, _ := cTransProofMap.Get(index)

	rops := message.RawOPS{
		ID:  content.Source,
		OPS: content.OPS,
	}
	transactions, _ := rops.Serialize()
	epochOPSMap.Insert(content.Seq, transactions)   //proposal(OPS) store here for state transfer
	epochMap.Insert(content.Seq-1, content.PreHash) //这里的qc(PreHash)是前一个seq的
	
	if tempres {
		p := fmt.Sprintf("I'm %v is in the trans committee, replying...", id)
		logging.PrintLog(verbose, logging.NormalLog, p)

		msg := message.ReplicaMessage{
			Mtype:   pb.MessageType_QCBCREP,
			Source:  id,
			Hash:    content.Hash,
			Seq:     content.Seq,
			PreHash: content.PreHash,
		}

		data, suc := ErasureEncoding(transactions, bucket.GetTBucketFailure()+1, bucket.GetKBucketNum())
		if !suc {
			log.Fatal("Fail to apply erasure coding during transmission!")
		}
		msg.TransactionRoot = GenMerkleTreeRoot(data) // accumulation value z
		serializedInfo := GenHashOfTwoVal(GenHashOfTwoVal(utils.Int64ToBytes(content.Source), msg.TransactionRoot), utils.IntToBytes(content.Seq))

		sig := cryptolib.GenSig(id, serializedInfo)
		msg.Signature = sig

		msg.VrfPi = VrfPi

		msgbyte, err := msg.Serialize()
		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[QCMessage Error] Not able to serialize the message")
			return
		}
		if content.Source == id {
			request, _ := message.SerializeWithSignature(id, msgbyte)
			HandleQCByteMsg(request)
		} else {
			go sender.SendToNode(msgbyte, content.Source)
		}

	}

	transStatus, _ := TransmissionStatus.Get(content.Seq)
	if epochOPSMap.GetLen(content.Seq) >= quorum.QuorumSize() && !transStatus { //check proposal是不是大于n-f
		TransmissionStatus.Insert(content.Seq, true)
		epochStatus.Insert(content.Seq-1, true)
		bcStatus.Set(READY)
		//quorum.ClearQC(content.Seq)

		if content.Seq >= 2 {
			trans_t2.Insert(content.Seq, utils.MakeTimestamp())
			log.Printf("catch trans_t2 for seq %v...", content.Seq)
			ExitTransmissionEpoch(content.Seq)
		}

		log.Printf("Transmission done, epochOPSMap len for seq %v is %v, epochMap len for seq %v is %v", content.Seq, epochOPSMap.GetLen(content.Seq), content.Seq-1, epochMap.GetLen(content.Seq-1))
	}

}

func HandleQCBCRepMessage(content message.ReplicaMessage) {
	status, exist := globalStatus.Get(content.Seq)
	if exist && status {
		return
	}

	serializedInfo := GenHashOfTwoVal(GenHashOfTwoVal(utils.Int64ToBytes(id), content.TransactionRoot), utils.IntToBytes(content.Seq))
	if !cryptolib.VerifySig(content.Source, serializedInfo, content.Signature) {
		p := fmt.Sprintf("[QC] signature for QCBCRep with height %v not verified", content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	commsg := message.ComMessage{
		Seq:     content.Seq,
		Process: "transmission",
		Source:  content.Source,
	}
	commsgbyte, _ := commsg.Serialize()
	totalReplicas := config.FetchNumReplicas()
	index := (totalReplicas * (content.Seq - 1)) + int(content.Source)

	if !ComProve(content.VrfPi, commsgbyte, index, "transmission") {
		p := fmt.Sprintf("[Transmission] vrfPi from node %v at epoch %v not verified", commsg.Source, content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	hash := utils.BytesToString(content.Hash)

	qclock.Lock()
	//quorum.Add(content.Source, hash, content.Signature, quorum.Pando, content.Seq)
	quorum.PandoAdd(content.Source, hash, content.Signature, quorum.Pando, content.Seq, content.VrfPi)
	status, _ = globalStatus.Get(content.Seq)
	if quorum.CheckBSmallQuorum(hash, content.Seq, quorum.Pando) && !status {
		globalStatus.Insert(content.Seq, true)
		qclock.Unlock()
		// log.Printf("[Transmission] Already get k-t signatures...")

		sigs, exist := quorum.FetchQC(content.Seq)
		ids, exist1 := quorum.FetchQCIDs(content.Seq)
		vrfs, exist2 := quorum.FetchQCVRFs(content.Seq)
		if !exist || !exist1 || !exist2 {
			p := fmt.Sprintf("[QCBC] cannnot obtain certificate from cache for block %v", content.Seq)
			logging.PrintLog(verbose, logging.NormalLog, p)
		}

		qcblock := message.QCBlock{
			Source: id,
			Height: content.Seq,
			QC:     sigs.Msgs,
			IDs:    ids,
			Hash:   serializedInfo,
			VRF:    vrfs.Msgs,
		}

		qcbytes, _ := qcblock.Serialize()
		curTransmissionQC.Insert(content.Seq, qcbytes)

		return
	}
	qclock.Unlock()

}

func HandleQCBCFinalMessage(content message.ReplicaMessage) {
	// blockinfo := message.DeserializeQCBlock(content.PreHash)
	log.Printf("[HandleQCBCFinalMessage] HandleQCBCFinalMessage todo message...")
}

func HandleQCPTwo(content message.ReplicaMessage) {
	// if evalMode > 0 {
	// 	evaluation(len(content.OPS), content.Seq)
	// }

	dstatus, _ := deliverStatus.Get(content.Seq)
	if dstatus {
		log.Printf("[HandleQCPTwo] This seq %v alreay delivered, process return.", content.Seq)
		return
	}

	if content.PreHash != nil {
		blockinfo := message.DeserializeQCBlock(content.PreHash) //QChigh
		if !VerifyBlock(content.Seq, blockinfo){
			p := fmt.Sprintf("[QC] QC Block %d not verified", blockinfo.Height)
			logging.PrintLog(true, logging.ErrorLog, p)
			return
		}
		if !VerifyPandoQC(blockinfo) {
			log.Printf("[HandleQCPTwo] Failed to verifyQC in seq %v, qcblock seq is %v.", content.Seq, blockinfo.Height)
			return
		}

		if !VerifyVrf(blockinfo, "consensus2") {
			log.Printf("[HandleQCPTwo] Failed to VRF in seq %v, consensus2 qcblock seq is %v.", content.Seq, blockinfo.Height)
			return
		}
	}

	hash := utils.BytesToString(content.Hash)
	v, _ := GetBufferContent(hash, BUFFER)
	if v == PREPARED {
		return
	}

	// if evalMode > 0 {
	// 	evaluation(len(content.OPS), content.Seq)
	// }

	if !CheckLeader(content.Seq, content.Source) {
		if verbose {
			log.Printf("[Replica] Received not from the leader, stop prepare in consensus process.")
			logging.PrintLog(verbose, logging.NormalLog, "[Replica] Received not from the leader, stop prepare in consensus process.")
		}
		return
	}
	log.Printf("[Prepare] Received msg from leader in seq %v", content.Seq)

	// //check if content.Hash is the hash in content
	// tmphash := CheckContentHash(content.GetMsgHash(), content.Seq)
	// tmphashstring := utils.BytesToString(tmphash)
	// if tmphashstring == hash {
	// 	storage.AddToStore(hash, content) //for obtainMissing
	// } else {
	// 	log.Printf("[CheckContentHash] Something wrong...")
	// }

	var tempres bool
	totalReplicas := config.FetchNumReplicas()
	index := (totalReplicas * (content.Seq - 1)) + int(id)
	tempres, _ = c2ConsIsCommitteeMap.Get(index)

	VrfPi, _ := c2ConsProofMap.Get(index)

	if tempres {
		p := fmt.Sprintf("I'm %v is in the committee 2, preparing...", id)
		logging.PrintLog(verbose, logging.NormalLog, p) //prepare start here.

		//ProcessQCInfo(hash, blockinfo, content)

		msg := message.ReplicaMessage{
			Mtype:  pb.MessageType_QCP3,
			Source: id,
			Hash:   content.Hash,
			Seq:    content.Seq,
		}
		//msg.PreHash = content.PreHash不用带了

		serializedInfo := GenHashOfTwoVal(GenHashOfTwoVal(content.Hash, utils.IntToBytes(1)), utils.IntToBytes(content.Seq))
		sig := cryptolib.GenSig(id, serializedInfo)
		msg.Signature = sig

		msg.VrfPi = VrfPi

		msgbyte, err := msg.Serialize()
		commitStatus.Insert(content.Seq, false)
		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[Replica Error] Not able to serialize the Prepare message.")
		} else {
			log.Printf("[Replica] Broadcast the Prepare message to all")
			logging.PrintLog(verbose, logging.NormalLog, "[Replica] Broadcast the Prepare message to all")
			/*dTime := utils.MakeTimestamp()
			diff,_ := utils.Int64ToInt(dTime - bTime)
			log.Printf("[%v] ++latency before sending %v ms", seq, diff)*/
			request, _ := message.SerializeWithSignature(id, msgbyte)
			HandleQCByteMsg(request)

			sender.RBCByteBroadcast(msgbyte)
		}
	} else {
		//ProcessQCInfo(hash, blockinfo, content)
		commitStatus.Insert(content.Seq, false)
	}

	UpdateBufferContent(hash, PREPARED, BUFFER)
}

func HandleQCPThree(content message.ReplicaMessage) {
	// hash := utils.BytesToString(content.Hash)
	// qc2lock.Lock()
	// v, _ := storage.GetBufferContent(hash, storage.BUFFER)
	// if v == storage.PREPARED{
	// 	qc2lock.Unlock()
	// 	return
	// }

	hash := utils.BytesToString(content.Hash)
	v, _ := GetBufferContent(hash, BUFFER)
	if v == COMMITTED {
		return
	}

	serializedInfo := GenHashOfTwoVal(GenHashOfTwoVal(content.Hash, utils.IntToBytes(1)), utils.IntToBytes(content.Seq))
	if !cryptolib.VerifySig(content.Source, serializedInfo, content.Signature) {
		p := fmt.Sprintf("[QC] signature for PrepareMessage with height %v not verified", content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	commsg := message.ComMessage{
		Seq:     content.Seq,
		Process: "consensus2",
		Source:  content.Source,
	}
	commsgbyte, _ := commsg.Serialize()
	totalReplicas := config.FetchNumReplicas()
	index := (totalReplicas * (content.Seq - 1)) + int(content.Source)

	if !ComProve(content.VrfPi, commsgbyte, index, "consensus2") {
		p := fmt.Sprintf("[Consensus2] vrfPi from node %v at epoch %v not verified", commsg.Source, content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	var tempres bool
	index = (totalReplicas * (content.Seq - 1)) + int(id)
	tempres, _ = c3ConsIsCommitteeMap.Get(index)

	qc2lock.Lock()
	//quorum.Add(content.Source, hash, content.Signature, quorum.PP, content.Seq)
	quorum.PandoAdd(content.Source, hash, content.Signature, quorum.PP, content.Seq, content.VrfPi)
	status, _ := commitStatus.Get(content.Seq)
	if quorum.CheckKQuorumSize(hash, content.Seq, quorum.PP) && !status {
		commitStatus.Insert(content.Seq, true)
		qc2lock.Unlock()
		UpdateBufferContent(hash, COMMITTED, BUFFER)

		// if content.Seq <= curBlock.Height {
		// 	return
		// }

		sigs, exist := quorum.FetchCer(content.Seq)
		ids, exist1 := quorum.FetchCerIDs(content.Seq)
		vrfs, exist2 := quorum.FetchCerVRFs(content.Seq)
		if !exist || !exist1 || !exist2 {
			p := fmt.Sprintf("[QC] cannnot obtain certificate from cache for block %v", content.Seq)
			logging.PrintLog(verbose, logging.NormalLog, p)
		}

		lockedqcblock := message.QCBlock{ //加上vrf
			Height: content.Seq,
			QC:     sigs.Msgs,
			IDs:    ids,
			Hash:   serializedInfo,
			VRF:    vrfs.Msgs,
		}
		curlockedBlock = lockedqcblock

		if !tempres {
			deliverStatus.Insert(content.Seq, false)
			//quorum.ClearPCer(content.Seq)

			return
		}

		p := fmt.Sprintf("I'm %v is in the committee 3, commiting...", id)
		logging.PrintLog(verbose, logging.NormalLog, p)

		msg := message.ReplicaMessage{
			Mtype:  pb.MessageType_QCP4,
			Seq:    content.Seq,
			Source: id,
			Hash:   content.Hash,
		}

		//msg.PreHash = content.PreHash

		serializedInfo := GenHashOfTwoVal(GenHashOfTwoVal(content.Hash, utils.IntToBytes(2)), utils.IntToBytes(content.Seq))
		sig2 := cryptolib.GenSig(id, serializedInfo)
		msg.Signature = sig2

		totalReplicas := config.FetchNumReplicas()
		index := (totalReplicas * (content.Seq - 1)) + int(id)
		VrfPi, _ := c3ConsProofMap.Get(index)
		msg.VrfPi = VrfPi

		msgbyte, _ := msg.Serialize()
		deliverStatus.Insert(content.Seq, false)
		//quorum.ClearPCer(content.Seq)

		sender.RBCByteBroadcast(msgbyte)
		log.Printf("[Replica] Broadcast the Commit message")
		logging.PrintLog(verbose, logging.NormalLog, "[Replica] Broadcast the Commit message")

		request, _ := message.SerializeWithSignature(id, msgbyte)
		HandleQCByteMsg(request)

		return
	}
	qc2lock.Unlock()
	// qc2lock.Unlock()
}

// deliver
func HandleQCPFour(content message.ReplicaMessage) {
	// hash := utils.BytesToString(content.Hash)
	// qc3lock.Lock()
	// v, _ := storage.GetBufferContent(hash, storage.BUFFER)
	// if v == storage.COMMITTED{
	// 	qc3lock.Unlock()
	// 	return
	// }

	hash := utils.BytesToString(content.Hash)
	serializedInfo := GenHashOfTwoVal(GenHashOfTwoVal(content.Hash, utils.IntToBytes(2)), utils.IntToBytes(content.Seq))
	if !cryptolib.VerifySig(content.Source, serializedInfo, content.Signature) {
		p := fmt.Sprintf("[QC] signature for CommitMessage with height %v not verified", content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	commsg := message.ComMessage{
		Seq:     content.Seq,
		Process: "consensus3",
		Source:  content.Source,
	}
	commsgbyte, _ := commsg.Serialize()
	totalReplicas := config.FetchNumReplicas()
	index := (totalReplicas * (content.Seq - 1)) + int(content.Source)

	if !ComProve(content.VrfPi, commsgbyte, index, "consensus3") {
		p := fmt.Sprintf("[Consensus3] vrfPi from node %v at epoch %v not verified", commsg.Source, content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	qc3lock.Lock()
	quorum.Add(content.Source, hash, content.Signature, quorum.CM, content.Seq)
	status, _ := deliverStatus.Get(content.Seq)
	if quorum.CheckKQuorumSize(hash, content.Seq, quorum.CM) && !status {
		deliverStatus.Insert(content.Seq, true)
		qc3lock.Unlock()

		log.Printf("[Replica] Deliver the message for seq %v", content.Seq)
		logging.PrintLog(verbose, logging.NormalLog, "[Replica] Deliver the message")

		sequence.UpdateLSeq(content.Seq)

		if content.Seq >= 1 {
			pando_t2.Insert(content.Seq+1, utils.MakeTimestamp())
			log.Printf("catch pando_t2 for seq %v...", content.Seq+1)
			ExitEpoch(content.Seq + 1)
		}

		curStatus.Set(READY)
		curNewViewStatus.Set(READY)
		quorum.ClearPCer(content.Seq)

		stateTransferHash.Insert(content.Seq, content.Hash)
		// tmpMsg, exist := storage.GetMsgFromMemoryByLeader(hash, CheckLeaderID(content.Seq))
		// if !exist {
		// 	log.Printf("[Deliver] Cannot retrival the message from leader %v in seq %v", CheckLeaderID(content.Seq), content.Seq)
		// } else {
		// 	if sequence.GetLEpoch()+1 != content.Seq {
		// 		FinalMsg := ObtainMissing(sequence.GetLEpoch()+1, content.Seq, tmpMsg)
		// 	} else {
		// 		FinalMsg := tmpMsg
		// 	}
		// }

		return
	}
	qc3lock.Unlock()
	//else if quorum.CheckKSQuorumSize(hash, content.Seq, quorum.CM) && !status {
	// 	v, _ := storage.GetBufferContent(hash, storage.BUFFER)
	// 	if v == storage.COMMITTED{
	// 		return
	// 	}
	// 	// storage.UpdateBufferContent(hash, storage.COMMITTED, storage.BUFFER)

	// 	var tempres bool
	// 	totalReplicas := config.FetchNumReplicas()
	// 	index := (totalReplicas*(content.Seq-1))+int(id)
	// 	tempres, _ = c3ConsIsCommitteeMap.Get(index)

	// 	if !tempres {
	// 		deliverStatus.Insert(content.Seq, false)
	// 		return
	// 	}

	// 	log.Printf("[Replica] Commit the message for speed up, send ready...")

	// 	msg := message.ReplicaMessage{
	// 		Mtype:   pb.MessageType_QCP4,
	// 		Seq:     content.Seq,
	// 		Source:  id,
	// 		Hash:    content.Hash,
	// 	}

	// 	serializedInfo := GenHashOfTwoVal(GenHashOfTwoVal(content.Hash, utils.IntToBytes(2)), utils.IntToBytes(content.Seq))
	// 	sig2 := cryptolib.GenSig(id, serializedInfo)
	// 	msg.Signature = sig2

	// 	totalReplicas = config.FetchNumReplicas()
	// 	index = (totalReplicas*(content.Seq-1))+int(id)
	// 	VrfPi, _ := c3ConsProofMap.Get(index)
	// 	msg.VrfPi = VrfPi

	// 	msgbyte, _ := msg.Serialize()
	// 	deliverStatus.Insert(content.Seq, false)

	// 	sender.RBCByteBroadcast(msgbyte)
	// 	log.Printf("[Replica] Broadcast the Commit message for speed up")
	// 	logging.PrintLog(verbose, logging.NormalLog, "[Replica] Broadcast the Commit message for speed up")

	// 	request, _ := message.SerializeWithSignature(id, msgbyte)
	// 	HandleQCByteMsg(request)

	// }

	// qc3lock.Unlock()
}

/*
Get data from buffer and cache. Used for consensus status
*/
func GetBufferContent(key string, btype TypeOfBuffer) (ConsensusStatus, bool) {
	switch btype {
	case BUFFER:
		v, exist := buffer.Get(key)
		return ConsensusStatus(v), exist
	}
	return 0, false
}

/*
Update buffer and cache. Used for consensus status
*/
func UpdateBufferContent(key string, value ConsensusStatus, btype TypeOfBuffer) {
	switch btype {
	case BUFFER:
		buffer.Insert(key, int(value))
	}
}

/*
Delete buffer and cache. Used for consensus status
*/
func DeleteBuffer(key string, btype TypeOfBuffer) {
	switch btype {
	case BUFFER:
		buffer.Delete(key)
	}
}
