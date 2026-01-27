package consensus

import (
	"log"
	"fmt"
	
	"pando/src/logging"
	"pando/src/config"
	"pando/src/cryptolib"
	"pando/src/message"
	"pando/src/utils"
	"pando/src/quorum"
	"pando/src/communication/sender"
	pb "pando/src/proto/communication"
	// sequence "pando/src/consensus/sequence"
)

func QCStartVC(){ //@QC
	log.Printf("start view change")
	vcTime = utils.MakeTimestamp() 
	curStatus.Set(VIEWCHANGE)
	newView = true 
	msg := message.ReplicaMessage{
		Mtype:    pb.MessageType_VIEWCHANGE,
		Source:   id,
		View:     LocalView(),
		TS:       utils.MakeTimestamp(),
		Num:      quorum.NSize(),
	}
    

	msg.Seq = curBlock.Height
	
	blockbyte,_ := curBlock.Serialize() 
	msg.PreHash = blockbyte 

	bbyte,_ := weakQC.Serialize()
	msg.Hash = bbyte

	awaitingBlocks.Init()
	awaitingDecision.Init()
	awaitingDecisionCopy.Init()

	msgbyte, err := msg.Serialize()

	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[QCVCMessage Error] Not able to serialize the message")
		return
	}

	curLeader := LeaderID(LocalView())
	cl := utils.IntToInt64(curLeader)
	p := fmt.Sprintf("[QC] qc starting view change to view %d sending qc-vc to %d", LocalView(), cl)
	logging.PrintLog(verbose, logging.NormalLog, p)

	sender.SendToNode(msgbyte, cl)
}


func PrepareLastVotes() []byte{
	mm := votedBlocks.GetAll() 
	if len(mm) == 0{
		return nil
	}
	var seqlist []int64
	var hashlist [][]byte
	for k,v := range mm{
		st := utils.IntToInt64(k)
		seqlist = append(seqlist,st)
		hashlist = append(hashlist,v)
	}

	m := message.QCBlock{
		QC:  hashlist,
		IDs: seqlist,
	}
	mb,_ := m.Serialize()
	return mb
}

func VerifyQC(qc message.QCBlock) bool{
	if qc.Hash == nil {
		return true 
	}
	
	if len(qc.QC) != len(qc.IDs) || len(qc.QC) == 0{
		return false 
	}
	
	//log.Printf("--[%v] length of QC %v", qc.Height, len(qc.QC))

	//t1 := utils.MakeTimestamp()
	for i:=0; i<len(qc.QC); i++{
		if !cryptolib.VerifySig(qc.IDs[i], qc.Hash, qc.QC[i]){
			p := fmt.Sprintf("[VerifyQC] signature not verified for height %v, source %v", qc.Height, qc.IDs[i])
			logging.PrintLog(true, logging.ErrorLog, p)
			return true
		}
	}

	//t2 := utils.MakeTimestamp()
	//log.Printf("time for verify QC %v", t2-t1)
	//log.Printf("[%v] signature verified ", qc.Height)
	return true
}

func VerifyPandoQC(qc message.QCBlock) bool {
	if qc.Hash == nil {
		return true 
	}
	
	if len(qc.QC) != len(qc.IDs) || len(qc.QC) == 0{
		return false 
	}
	
	if len(qc.QC) < quorum.KQuorumSize() {
		p := fmt.Sprintf("[VerifyQC] Pando signature size smaller than k-t")
		logging.PrintLog(true, logging.ErrorLog, p)			
		return true
	}
	return true
}

func VerifyVrf(qc message.QCBlock, process string) bool {
	if qc.Hash == nil {
		return true 
	}
	
	if len(qc.VRF) != len(qc.IDs) || len(qc.VRF) == 0{
		return false 
	}
	
	totalReplicas := config.FetchNumReplicas()
	for i:=0; i<len(qc.VRF); i++{
		commsg := message.ComMessage{
			Seq:		qc.Height,
			Process:	process,
			Source:		qc.IDs[i],
		}
		commsgbyte, _ := commsg.Serialize()
		index := (totalReplicas*(commsg.Seq-1))+int(commsg.Source)
	
		if !ComProve(qc.VRF[i], commsgbyte, index, commsg.Process) {
			p := fmt.Sprintf("[VerifyVrf] vrf not verified for height %v", qc.Height)
			logging.PrintLog(true, logging.ErrorLog, p)
			return true
		}
	}

	return true
}

// func HandleQCVCMessage(content message.ReplicaMessage, sigversion int, vcm message.MessageWithSignature) { //For new leader to collect vc messages. Todo: double check VC rules @QC
	
// 	if content.View != LocalView() && content.View != LocalView() + 1{
// 		p := fmt.Sprintf("[QC] Handle view change to view %d from %v, local view %d", content.View, content.Source, LocalView())
// 		logging.PrintLog(true, logging.ErrorLog, p)
// 		return 
// 	}
// 	tmp, _ := utils.Int64ToInt(id)
// 	if LeaderID(content.View) != tmp{
// 		return 
// 	}

// 	cb := message.DeserializeQCBlock(content.PreHash)

// 	if !VerifyQC(cb){
// 		log.Printf("qc not verified in QCVC %v", content.View)
// 		return 
// 	}
	
// 	if cb.Hash !=nil && cb.Height >= curBlock.Height{
// 		curBlock = cb 
// 		sequence.UpdateLSeq(cb.Height)
// 		vs,_ := vcAwaitingVotes.Get(content.View)
// 		if cb.Height > vs{
// 			vcAwaitingVotes.Delete(content.View)
// 		}
// 	}

// 	if config.QCMode() == 0 {
// 		lb := message.DeserializeQCBlock(content.Hash) 
// 		//log.Printf("[%v] last block height", lb.Height)
// 		vcTracker.Increment(lb.Height)
// 		tmp,_ := vcTracker.Get(lb.Height)

// 		if tmp >= quorum.SQuorumSize(){
// 			vcHashTracker.Insert(lb.Height,lb.Hash)
// 			//log.Printf("---------update height from %v to %v", sequence.GetSeq(), lb.Height)
// 			sequence.UpdateSeq(lb.Height)
// 		}
// 	}
	
// 	//quorum.Add(content.Source, hash, content.PreHash, quorum.PP, content.View)
	
// 	quorum.AddToIntBuffer(content.View, content.Source, vcm, quorum.VC)

// 	//if quorum.CheckQuorum(hash, content.View, quorum.PP) {
// 	hash := utils.BytesToString(cryptolib.GenHash(utils.IntToBytes(content.View)))
// 	vcblock.Lock()
// 	v, _ := GetBufferContent(hash, BUFFER)
// 	if v == PREPARED{
// 		vcblock.Unlock()
// 		return 
// 	}

// 	if curStatus.Get() == READY{
// 		vcblock.Unlock()
// 		return 
// 	}
		

// 	if quorum.CheckIntQuorum(content.View, quorum.VC){
// 		SetView(content.View)
// 		UpdateBufferContent(hash, PREPARED, BUFFER)
		
// 		switch config.Consensus(){
// 		default:
// 			curStatus.Set(READY)
// 			//timer.Stop()
// 			//HandleCachedMsg()
// 			go RequestMonitor()
// 		}
		
// 	}
// 	vcblock.Unlock()
// }


func GetQCOpsfromV(VV []message.MessageWithSignature, verify bool) map[int]message.MessageWithSignature{
	o := make(map[int]message.MessageWithSignature)
	min := 1<<(UintSize-1) - 1 // set min to inifinity
	max := 0
	
	for i := 0; i < len(VV); i++ {
		V := message.DeserializeViewChangeMessage(VV[i].Msg)
		pCer := V.P
		for k, v := range pCer {
			msgs := v.GetMsgs()[0]
			bl := message.DeserializeQCBlock(msgs)
			//log.Printf("k %v, height %v, hash %s", k, bl.Height, bl.Hash)
			_, exist := o[k]
			if exist {
				continue
			}
			if !VerifyQC(bl){
				continue 
			}
			if k < min {
				min = k
			}
			if k > max {
				max = k
			}

			tmpmsg := message.ReplicaMessage{
				Mtype:   pb.MessageType_QC2,
				Seq:     k,
				Source:  id,
				View:    view,
				Hash:    bl.Hash,
				PreHash: msgs,
			}
			op := message.CreateMessageWithSig(tmpmsg)

			o[k] = op
		}
		
	}
	
	return o 
}

// func HandleQCNewView(rawmsg []byte){
// 	vcm := message.DeserializeViewChangeMessage(rawmsg)

// 	nlID, _ := utils.Int64ToInt(id)
// 	if LeaderID(vcm.View) == nlID {
// 		p := fmt.Sprintf("[View Change] Replica %d becomes leader", id)
// 		logging.PrintLog(verbose, logging.NormalLog, p)
// 		leader = true
// 	} else {
// 		leader = false
// 	}
	
// 	curStatus.Set(READY)
// 	//timer.Stop()
// 	OPS := vcm.O
// 	for k, rm := range OPS {
// 		msg, err := rm.Serialize()
// 		if err != nil {
// 			p := fmt.Sprintf("[New View Error] Serialize the PRE-PREPARE message of OPS in the NEW-VIEW message failed: %v", err)
// 			logging.PrintLog(true, logging.ErrorLog, p)
// 		}

// 		p := fmt.Sprintf("[New View] handle PP from new view, seq = %d", k)
// 		logging.PrintLog(verbose, logging.NormalLog, p)
		
// 		HandleQCByteMsg(msg)
// 	}
// }

// func HandleSecondQCVCMessage(content message.ReplicaMessage) {
	
// 	if content.View != LocalView() && content.View != LocalView() + 1{
// 		return 
// 	}


// 	msg := message.ReplicaMessage{
// 		Mtype:    pb.MessageType_VIEWCHANGE2REP,
// 		Source:   id,
// 		View:     LocalView(),
// 		TS:       utils.MakeTimestamp(),
// 		Num:      quorum.NSize(),
// 	}
    
// 	if curBlock.Height >= content.Seq {
// 		blockbyte,_ := curBlock.Serialize() 
// 		msg.PreHash = blockbyte 
// 		msg.Seq = curBlock.Height 
		
// 	}else{
// 		cb := message.DeserializeQCBlock(content.PreHash)
// 		sig := cryptolib.GenSig(id,cb.Hash)
// 		msg.Hash = cb.Hash
// 		msg.Seq = content.Seq 
// 		msg.PreHash = sig
// 	}
	
	

// 	msgbyte, err := msg.Serialize()

// 	if err != nil {
// 		logging.PrintLog(true, logging.ErrorLog, "[QCVCMessage Error] Not able to serialize the message")
// 		return
// 	}

// 	curLeader := LeaderID(content.View)
// 	cl := utils.IntToInt64(curLeader)
	
// 	p := fmt.Sprintf("[QC] handling viewchange2 message from %v in view %d, %v", content.Source, LocalView(), cl)
// 	logging.PrintLog(verbose, logging.NormalLog, p)
// 	log.Printf("sending viewchange2 message to view %v, leader %v", content.View, cl)
	
// 	sender.SendToNode(msgbyte, cl)

// }



// func HandleSecondQCVCRepMessage(content message.ReplicaMessage, sigversion int, vcm message.MessageWithSignature) {
// 	//log.Printf("handling 2nd qcvc from %v", content.Source)
// 	if curStatus.Get() == READY{
// 		p := fmt.Sprintf("[QC] not in view change status %d", LocalView())
// 		logging.PrintLog(verbose, logging.NormalLog, p)
// 		return 
// 	}
// 	if content.View != LocalView() && content.View != LocalView() + 1{
// 		//log.Printf("contentView %v, localview %v", content.View, LocalView())
// 		return 
// 	}
// 	tmp, _ := utils.Int64ToInt(id)
// 	if LeaderID(content.View) != tmp{
// 		return 
// 	}
	
// 	if content.Seq > curBlock.Height && content.PreHash != nil{
// 		cb := message.DeserializeQCBlock(content.PreHash)
// 		if !VerifyQC(cb,sigversion){
// 			return
// 		}
// 		if cb.Hash !=nil && cb.Height > curBlock.Height{
// 			curBlock = cb
// 			vs,_ := vcAwaitingVotes.Get(content.View)
// 			if curBlock.Height ==  vs {
// 				//curStatus.Set(READY)
// 				timer.Stop()
// 				HandleCachedMsg()
// 				go RequestMonitor()
// 				return 
// 			}
// 		}
// 	}

// 	if content.Seq < curBlock.Height{
// 		log.Printf("content seq %v, curblock height %v", content.Seq, curBlock.Height)
// 		return 
// 	}

// 	if content.Hash != nil && !cryptolib.VerifySig(content.Source, content.Hash, content.PreHash){
// 		log.Printf("sig not verified")
// 		return 
// 	}
// 	//todo: verify QC

// 	//log.Printf("proceed 2nd qvcbc")
// 	vclock.Lock()
// 	if curStatus.Get() == READY{
// 		vclock.Unlock()
// 		return 
// 	}
// 	hash := utils.BytesToString(content.Hash)
// 	quorum.Add(content.Source, hash, content.PreHash, quorum.PP,content.Seq)
// 	if quorum.CheckQuorum(hash, content.Seq, quorum.PP) {
// 		sigs,exist := quorum.FetchCer(content.Seq)
// 		ids,exist1 := quorum.FetchCerIDs(content.Seq)
// 		if !exist || !exist1{
// 			log.Printf("error! cannot obtain certificate from cache")
// 		}
		
// 		qcblock := message.QCBlock{
// 			View:		LocalView(),
// 			Height:		content.Seq, 
// 			QC:			sigs.Msgs, 
// 			IDs:		ids, 	
// 			Hash:       content.Hash,
// 		}
// 		log.Printf("received a quorum of messages in view %v, quorum size %v, done", LocalView(), quorum.QuorumSize())
// 		vcBlock = qcblock 
// 		curStatus.Set(READY) 
// 		log.Printf("status ready %v=%v, queue %v, queuelen %v", sequence.GetSeq(), lastSeq, queue.IsEmpty(), awaitingDecisionCopy.GetLen())
// 		//timer.Stop()
// 		go RequestMonitor()
// 		leaderVCstatus = true
// 		go HandleCachedMsg()
// 	}
// 	vclock.Unlock()
		
// }

// func HandleCachedMsg(){
// 	msgnvlist := cachedMsgForNextView.Grab()
// 	cachedMsgForNextView.Clear()
// 	for i := 0; i < len(msgnvlist); i++ {
// 		HandleQCByteMsg(msgnvlist[i].GetMsg())
// 	}
// }
