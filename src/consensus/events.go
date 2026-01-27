package consensus

import (
	"fmt"
	"log"
	"pando/src/config"
	"pando/src/logging"
	// "pando/src/quorum"
	"pando/src/utils"
	"time"
	"github.com/vmihailenco/msgpack/v5"

	pb "pando/src/proto/communication"
	sender "pando/src/communication/sender"
	message "pando/src/message"
	sequence "pando/src/consensus/sequence"
)

var verbose bool //verbose level
var id int64     //id of server
var iid int      //id in type int, start a RBC using it to instanceid
var errs error
var queue Queue         // cached client requests
var queueHead QueueHead // hash of the request that is in the first place of the queue
var sleepTimerValue int // sleeptimer for the while loop that continues to monitor the queue or the request status
var consensus ConsensusType
var rbcType RbcType
var n int
var members []int
var t1 int64
var baseinstance int

var requestSize int

var midTime map[int]int64

var MsgQueue Queue // record the consensus messages received so far.


func CaptureRBCLat() {
	t3 := utils.MakeTimestamp()
	if (t3 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	log.Printf("*****RBC phase ends with %v ms", t3-t1)

}

func CaptureLastRBCLat() {
	t3 := utils.MakeTimestamp()
	if (t3 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	log.Printf("*****Final RBC phase ends with %v ms", t3-t1)

}

// This func is used to grab cached requests from clients and propose new proposals.
// So if the node is not a leader, it will leave this func.
// Modify: when invoking this func, users need to input the view number associated with this monitor.
// This monitor will return when its view number is not the current view.
func RequestMonitor() {
	epoch := 0
	status := false
	time.Sleep(time.Duration(5 * time.Second))

	for {
		// if v != LocalView() {
		// 	return
		// }
		switch consensus {

		case Pando:
			epoch = sequence.GetLEpoch()	//epoch=LEpoch=1 when server start
			log.Printf("############### epoch %v...", epoch)				
			
			if !CheckLeader(epoch, id) {
				time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
				continue			
			}	

			status, _ = epochStatus.Get(epoch) //|| curStatus.Get() == QCTimeout, awaitingDecisionCopy.GetLen()>0||
			if !((curStatus.Get() == READY) && (status)) {
				time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
				continue
			}

			if curConsensusStatus.Get() != READY {
				time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
				continue
			}

			seq := curStatusSeq
			log.Printf("I'm consensus leader ready for seq %v, start processing...",seq)

			dstatus,_ := deliverStatus.Get(seq)
			cstatus,_ := commitStatus.Get(seq)
			if dstatus || cstatus{
				continue
			}	

			curStatus.Set(PROCESSING)	
			curConsensusStatus.Set(PROCESSING)

			var batch []pb.RawMessage	
			if curBlock.Height+1 < epoch {
				log.Printf("[QC] Currect qchigh height %v is smaller than epoch %v.", curBlock.Height, epoch)
				for i := curBlock.Height+1; i <= epoch; i++ {
					setofMsgs, _ := epochMap.Get(i)
					for j := 0; j < len(setofMsgs); j++ {
						tmp := pb.RawMessage{
							Msg: setofMsgs[j],
						}
						batch = append(batch, tmp)
					}
				}
			}else {
				setofMsgs, _ := epochMap.Get(epoch)
				for i := 0; i < len(setofMsgs); i++ {
					tmp := pb.RawMessage{
						Msg: setofMsgs[i],
					}
					batch = append(batch, tmp)
				}					
			}

			msg := message.ReplicaMessage{
				Mtype:    pb.MessageType_QCP2,
				Seq:      seq,
				Source:   id,
				OPS:      batch,	
				TS:       utils.MakeTimestamp(),
			}
			
			msg.PreHash = FetchBlockInfo(seq)//PreHash=curBlock.Serialize()
			tmphash := msg.GetMsgHash() 
			msg.Hash = ObtainCurHash(tmphash, msg.Seq)

			preHash.Set(curHash.Get())
			curHash.Set(msg.Hash)

			msgbyte, err := msg.Serialize()
			//quorum.ClearPCer(content.Seq)

			if err != nil {
				logging.PrintLog(true, logging.ErrorLog, "[Replica Error] Not able to serialize the Propose message.")
			} else {
				log.Printf("Leader broadcast the Propose message to all")
				logging.PrintLog(verbose, logging.NormalLog, "[Replica] Leader broadcast the Propose message to all")
				request, _ := message.SerializeWithSignature(id, msgbyte)
				HandleQCByteMsg(request)

				sender.RBCByteBroadcast(msgbyte)
			}
		}

		continue

	}
}

func HandleRequest(request []byte, hash string) {
	//log.Printf("Handling request")
	//rawMessage := message.DeserializeMessageWithSignature(request)
	//m := message.DeserializeClientRequest(rawMessage.Msg)

	/*if !cryptolib.VerifySig(m.ID, rawMessage.Msg, rawMessage.Sig) {
		log.Printf("[Authentication Error] The signature of client request has not been verified.")
		return
	}*/
	//log.Printf("Receive len %v op %v\n",len(request),m.OP)
	// batchSize = 1
	requestSize = len(request)
	queue.Append(request)
}


func DeserializeRequests(input []byte) [][]byte {
	var requestArr [][]byte
	msgpack.Unmarshal(input, &requestArr)
	return requestArr
}


func NewViewMonitor() {
	time.Sleep(time.Duration(5 * time.Second))

	for {
		if !(curNewViewStatus.Get() == READY) {	
			time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
			continue
		}

		//TODO: Set timer 判断前一轮是否超时，超时则NewView进入下一epoch

		epoch := sequence.GetLEpoch()
		if epoch > 0 {
			epoch = sequence.EpochLIncrement()
			for {
				status, _ := epochStatus.Get(epoch)
				dstatus,_ := deliverStatus.Get(epoch-1)
				if (status) && (dstatus) && (curNewViewStatus.Get() == READY) {
					break
				}
				time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
			}
		} else if epoch == 0 {
			epoch = sequence.EpochLIncrement()
			for {
				status, _ := epochStatus.Get(epoch)
				if (status) && (curNewViewStatus.Get() == READY) {
					break
				}
				time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
			}
		}
	
		// epoch := sequence.GetLEpoch()	//epoch=LEpoch=1 when server start
		// status, _ := epochStatus.Get(epoch) //|| curStatus.Get() == QCTimeout, awaitingDecisionCopy.GetLen()>0||

		// if !(status && curNewViewStatus.Get() == READY) {	
		// 	time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
		// 	continue
		// }

		dstatus,_ := deliverStatus.Get(epoch)
		cstatus,_ := commitStatus.Get(epoch)
		if dstatus || cstatus{
			continue
		}

		curNewViewStatus.Set(PROCESSING)
		log.Printf("Consensus newview processing %v...", epoch)

		if epoch >= 1{
			cons_t1.Insert(epoch+1, utils.MakeTimestamp())
			log.Printf("catch cons_t1 for seq %v...", epoch+1)
		}

		var tempres bool
		totalReplicas := config.FetchNumReplicas()
		index := (totalReplicas*(epoch-1))+int(id)
		tempres, _ = c1ConsIsCommitteeMap.Get(index)

		VrfPi, _ := c1ConsProofMap.Get(index)

		if tempres{
			//sequence.EpochLIncrement()
			seq := sequence.IncrementLSeq()
			p := fmt.Sprintf("I'm %v is in the committee 1, sending newview...", id)
			logging.PrintLog(verbose, logging.NormalLog, p)

			msg := message.ReplicaMessage{
				Mtype:    pb.MessageType_NEWVIEW,
				Seq:      seq,
				Source:   id,
				TS:       utils.MakeTimestamp(),
			}

			if msg.Seq > 1 {
				msg.PreHash = FetchLockedBlockInfo(msg.Seq-1)
			}

			msg.VrfPi = VrfPi
			NVcounter.Insert(msg.Seq, 0)

			msgbyte, err := msg.Serialize()
			if err != nil {
				logging.PrintLog(true, logging.ErrorLog, "[Replica Error] Not able to serialize the NewView message.")
			} else {
				consensusLeader := CheckLeaderID(msg.Seq)
				log.Printf("Send the NewView message to the leader %v in seq %v", consensusLeader, msg.Seq)
				logging.PrintLog(verbose, logging.NormalLog, "[Replica] Send the NewView message to the leader")
				
				if CheckLeader(msg.Seq, id) {
					request, _ := message.SerializeWithSignature(id, msgbyte)
					HandleQCByteMsg(request)
				} else {
					sender.SendToNode(msgbyte, consensusLeader)
				}
			}

		}
	}

}

// func RetrievalMonitor() {
// 	for {
// 		if !(curStateTransStatus.Get() == READY) {	
// 			time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
// 			continue
// 		}			

// 		epoch := sequence.Increment()
		
// 		var stHash []byte
// 		for {
// 			status, _ := epochStatus.Get(epoch)
// 			dstatus,_ := deliverStatus.Get(epoch)
// 			stHash, _ = stateTransferHash.Get(epoch)
// 			if status && dstatus && stHash != nil {
// 				break
// 			}
// 			time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)		
// 		}
// 		//state_t1.Insert(epoch, utils.MakeTimestamp())	//TODO:how to set t1? sometimes received frags before send distribute.
// 		tmpt,_ := state_t1.Get(epoch)
// 		if tmpt == 0 {
// 			state_t1.Insert(epoch, utils.MakeTimestamp())
// 		}	

// 		var consensusPropose []pb.RawMessage
// 		for {
// 			tmpMsg, exist := storage.GetMsgFromMemory(utils.BytesToString(stHash))
// 			if exist {
// 				consensusPropose = tmpMsg.OPS	//msg, len(consensusPropose)=n
// 				break
// 			}
// 			time.Sleep(time.Duration(5*sleepTimerValue) * time.Millisecond)
// 		}

// 		curStateTransStatus.Set(PROCESSING)
// 		log.Printf("State transfer processing...")
// 		//state_t1.Insert(epoch, utils.MakeTimestamp())

// 		var tempres bool
// 		totalReplicas := config.FetchNumReplicas()
// 		index := (totalReplicas*(epoch-1))+int(id)
// 		tempres, _ = c1StateIsCommitteeMap.Get(index)
// 		VrfPi, _ := c1StateProofMap.Get(index)

// 		setofTrans, _ := epochOPSMap.Get(epoch)		//rops, proposal, len(setofTrans)=n, contains transactionRoot in transmission
// 		var awaitingProposal []message.RawOPS
// 		for proposal := 0; proposal < len(setofTrans); proposal++ {
// 			rops := message.DeserializeRawOPS(setofTrans[proposal])		//message.RawOPS
// 			//log.Printf("rops is this epoch is %v...\n%v", rops.ID, rops.OPS)
// 			for i := 0; i < len(consensusPropose); i++ {
// 				qcblockinfo := message.DeserializeQCBlock(consensusPropose[i].Msg)		//message.QCBlock
// 				if qcblockinfo.Source != rops.ID {
// 					continue
// 				}
// 				//log.Printf("[StateTransfer] retrieval qcblock for node %v...", qcblockinfo.Source)
// 				awaitingProposal = append(awaitingProposal, rops)

// 				stStatus, _ := retrievalEnd.Get(epoch)
// 				if tempres && !stStatus {
// 					transactions, _ := rops.Serialize()	
// 					data, suc := ErasureEncoding(transactions, bucket.GetTBucketFailure()+1, bucket.GetKBucketNum())	//len(data)=k
// 					if !suc{
// 						log.Printf("[Distribute] Fail to apply erasure coding during state transfer!")
// 						return
// 					}
// 					transactionRoot := GenMerkleTreeRoot(data)
// 					// serializedInfo := GenHashOfTwoVal(GenHashOfTwoVal(utils.Int64ToBytes(rops.ID), transactionRoot), utils.IntToBytes(epoch))
// 					// if !cryptolib.VerifySig(rops.ID, sigversion, serializedInfo, qcblockinfo.QC[i], epoch){
// 					// 	p := fmt.Sprintf("[ST] signature of QC is not verified")
// 					// 	logging.PrintLog(true, logging.ErrorLog, p)
// 					// 	return 
// 					// }			
// 					branches, idxresult := ObtainMerklePath(data) 	//need to use idxresult?	 （branches有k个frag，len(frag)=log k)
// 					// TestVerifyMerkleRoot(transactionRoot, data[0], branches[0], idxresult[0])
					
// 					for k := 0; k < bucket.GetKBucketNum(); k++{
// 						msg := message.ReplicaMessage{
// 							Mtype:				pb.MessageType_QC,		//go HandleDistribute()
// 							Seq:				epoch,
// 							Source:				id,
// 							Num:				k+1,
// 							TransactionRoot:	transactionRoot,
// 							Witness:			branches[k],
// 							Fragment:			data[k],
// 							Index:				idxresult[k],
// 							VrfPi:				VrfPi,
// 						}

// 						// serializedInfo := GenHashOfTwoVal(GenHashOfTwoVal(utils.Int64ToBytes(id), msg.TransactionRoot), utils.IntToBytes(content.Seq))
// 						// sig := cryptolib.GenSig(id, serializedInfo)
// 						// msg.Signature = sig				

// 						msgbyte, err := msg.Serialize()
// 						if err != nil {
// 							logging.PrintLog(true, logging.ErrorLog, "[StateTransfer Error] Not able to serialize the message")
// 							return
// 						}

// 						//log.Printf("Send the Distribute message to bucket node")
// 						logging.PrintLog(verbose, logging.NormalLog, "[Replica] Send the Distribute message to bucket node")												
// 						for j := 0; j < totalReplicas; j++ {
// 							temp, _ := cIsBucketMap.Get(j)
// 							if temp == k+1 {
// 								if int64(j) == id{
// 									request, _ := message.SerializeWithSignature(id, msgbyte)
// 									HandleQCByteMsg(request)														
// 								}else {
// 									go sender.SendToNode(msgbyte, int64(j))
// 								}
								
// 							}
// 						}
// 					}

// 				}


// 			}
// 		}

// 		// stStatus, _ := retrievalEnd.Get(epoch)
// 		// if len(awaitingProposal) == len(consensusPropose) && stStatus {
// 		// 	log.Fatal("[StateTransfer] awaitingProposal size equals consensusPropose size, done state transfer...")
					
// 		// 	curStateTransStatus.Set(READY)
// 		// }

// 	}
// }



// // Monitor the queue and start timer if the queue is not empty
// //QCP
// func TimerMonitor() {
// 	if config.Consensus() == config.QCP || config.Consensus() == config.Pando {
// 		time.Sleep(time.Duration(5 * time.Second))
// 	}
// 	//looping execution
// 	for {

// 		if rotatingLeader {
// 			if joinStatus == COMPLETED && queue.Length() > 0 && awaitingDecision.GetLen() > 0 { //curStatus.Get() == READY{
// 				//log.Printf("start timer")
// 				timer = time.AfterFunc(time.Duration(timerValue)*time.Millisecond, ExpireEvent)
// 				return
// 			} else {
// 				time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
// 			}
// 		} else {
// 			if config.Consensus() == config.QCP {
// 				curEpoch := sequence.GetLEpoch() + 1
// 				if !Leader() && lastVCEpoch.Get() != curEpoch {
// 					curStatus, _ := epochStatus.Get(curEpoch)

// 					if curStatus { //&& lastVCEpoch.Get() == curEpoch{
// 						lastVCEpoch.Set(curEpoch)
// 						timer = time.AfterFunc(time.Duration(timerValue)*time.Millisecond, ExpireEvent)
// 					} else {
// 						time.Sleep(time.Duration(10*sleepTimerValue) * time.Millisecond)
// 					}
// 				}

// 			}

// 			if config.Consensus() == config.Pando {
// 				curEpoch := sequence.GetLEpoch() + 1
// 				if !CheckLeader(curEpoch, id) && lastVCEpoch.Get() != curEpoch {
// 					curStatus, _ := epochStatus.Get(curEpoch)
// 					if curStatus {
// 						lastVCEpoch.Set(curEpoch)
// 						timer = time.AfterFunc(time.Duration(timerValue)*time.Millisecond, ExpireEvent)						
// 					}else {
// 						time.Sleep(time.Duration(10*sleepTimerValue) * time.Millisecond)
// 					}
// 				}

// 			}
// 		}
// 	}
// }
