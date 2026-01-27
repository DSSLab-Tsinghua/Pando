/*
This file implements functions for evaluating the performance of the protocol
*/

package consensus

import (
	"log"
	"fmt"
	"sync"

	"pando/src/cryptolib"
	"pando/src/logging"
	"pando/src/message"
	"pando/src/utils"
	"pando/src/quorum"
	"github.com/vmihailenco/msgpack/v5"
)


var curOPS utils.IntValue
var totalOPS utils.IntValue
var beginTime int64
var lastTime int64
var clock sync.Mutex
var bTime int64 
var batchSize utils.IntIntMap
var cons_t1 utils.IntInt64Map
var trans_t2 utils.IntInt64Map
var pando_t1 utils.IntInt64Map	//the beginning of transmission
var pando_t2 utils.IntInt64Map	//the end of consensus

var state_t1 utils.IntInt64Map
var state_t2 utils.IntInt64Map

var evalInterval int

// Deserialize bytes to an array of client reuests.
func DeserializeClientBatch(input []byte) [][]byte{
	var clientRequests [][]byte
	msgpack.Unmarshal(input, &clientRequests)
	return clientRequests
}


//For evaluation only. The way how hashes are generated cannot enforce all requests will be replied.
func GetMsgs(request []byte) ([][]byte, []string){
	msgs := DeserializeClientBatch(request)
	var hashes []string
	for i:=0; i<len(msgs); i++{
		//msg,_:= msgs[i].Serialize()
		hash := string(cryptolib.GenHash(msgs[i]))
		hashes = append(hashes, hash)
	}
	return msgs, hashes
}

// VerifySignature
//	@Description: verify digital signature
//QCP
func VerifySignature(id int64, version int, msg []byte, sig []byte, seq int) bool {
	verifyFlag := cryptolib.VerifySig(id, msg, sig)	//verifyFlag=true is pass crypto Verify
	if !verifyFlag {
		// permission.ObtainConfigAndPubKeys(utils.Int64ToString(id))
		verifyFlag = cryptolib.VerifySig(id, msg, sig)
	}
	return verifyFlag
}

	
// //Handle batch client requests
// func HandleBatchRequest(msgs []byte, hash string){
// 	rawMessage := message.DeserializeMessageWithSignature(msgs)
// 	m := message.DeserializeClientRequest(rawMessage.Msg)
// 	if !VerifySignature(m.ID, 0, rawMessage.Msg, rawMessage.Sig, 0) {
// 		p := fmt.Sprintf("[Authentication Error] The signature of consortium client [%v] registration/key reset request has not been verified.", m.ID)
// 		logging.PrintLog(true, logging.ErrorLog, p)
// 		// storage.InsertInvalid(hash)
// 		return
// 	}

// 	// if (!storage.IsDelivered(hash)){
// 	// 	//msg,_ := msgs.Serialize()
// 	// 	queue.Append(msgs)
// 	// }
// }


//Send default reply message to the clients via channel. Used for evaluation only.
func GetTempResponseViaChan(result chan []byte)  {
	//for {
		//if (curStatus.Get()==READY){
			cr := message.ClientReply{
				Source: id,  
				Msg: []byte("done"),
			}
			r,_ := cr.Serialize()

			rr := r
		
			result <- rr 
		//}
	//}
}

func evalStep(){
	curtime := utils.MakeTimestamp()
	diff := curtime - lastTime 
	lastTime = curtime 
	log.Printf("this step time %v ms", diff)
}

// Get the throughput of the system
//QCP
func evaluation(lenOPS int, seq int){
	if lenOPS > 1 {
		//log.Printf("[Replica] evaluation mode with %d ops", lenOPS)
		var p = fmt.Sprintf("[Replica] evaluation mode with %d ops", lenOPS)
		logging.PrintLog(verbose, logging.EvaluationLog, p)
	}
	
	clock.Lock()
	defer clock.Unlock()
	val := curOPS.Get()
	if  seq == 1{
		beginTime = utils.MakeTimestamp()
		lastTime = beginTime
		// genesisTime = utils.MakeTimestamp()
	}

	tval := totalOPS.Get()
	lenOPS = lenOPS //* 5
	tval = tval + lenOPS
	totalOPS.Set(tval)

	if val+lenOPS >= evalInterval{
		curOPS.Set(0)
		var endTime = utils.MakeTimestamp()
		var throughput int
		lat,_ := utils.Int64ToInt(endTime - beginTime)
		if lat > 0{
			throughput = 1000*(val+lenOPS)/lat
		}
		
		// clockTime, _ := utils.Int64ToInt(utils.MakeTimestamp() - genesisTime)
		// log.Printf("[Replica] Processed %d (ops=%d, clockTime=%d ms, seq=%v) operations using %d ms. Throughput %d tx/s. ", tval, lenOPS, clockTime, seq, lat, throughput)
		// var p = fmt.Sprintf("[Replica] Processed %d (ops=%d, clockTime=%d ms, seq=%v) operations using %d ms. Throughput %d tx/s", tval, lenOPS, clockTime, seq, lat, throughput)
		log.Printf("[Replica] Processed %d (ops=%d, seq=%v) operations using %d ms. Throughput %d tx/s. ", tval, lenOPS, seq, lat, throughput)
		var p = fmt.Sprintf("[Replica] Processed %d operations using %d ms. Throughput %d tx/s", tval, lat, throughput)
		logging.PrintLog(true, logging.EvaluationLog, p)
		beginTime = endTime
		lastTime = beginTime
		
	}else{
		curOPS.Set(val+lenOPS)
	}
}

func ExitEpoch(cur_epoch int){
	if cur_epoch <= 1 {
		return
	}

	clock.Lock()
	tmpt1,_ := pando_t1.Get(cur_epoch)
	tmpt2,_ := pando_t2.Get(cur_epoch)
	//tmptrans_t2,_ := trans_t2.Get(cur_epoch)
	tmpcons_t1,_ := cons_t1.Get(cur_epoch)
	tmpbatchsize,_ := batchSize.Get(cur_epoch)
	clock.Unlock()

	if tmpt2-tmpcons_t1 == 0 || tmpt2 == 0 || tmpcons_t1 == 0 {
		return
	}

	if tmpt2-tmpt1 == 0 || tmpt1 == 0 {
		return
	}

	log.Printf("[Consensus] epoch %v ends with batchsize %v, Consensus latency %v ms, tps %d", cur_epoch, tmpbatchsize, tmpt2-tmpcons_t1, int64(quorum.QuorumSize()*tmpbatchsize*1000)/(tmpt2-tmpcons_t1))
	p := fmt.Sprintf("[Consensus] %v %v %v %d",cur_epoch, tmpbatchsize, tmpt2-tmpcons_t1, int64(quorum.QuorumSize()*tmpbatchsize*1000)/(tmpt2-tmpcons_t1))
	logging.PrintLog(true,logging.EvaluationLog,p)

	log.Printf("[Total] epoch %v ends with batchsize %v, Trans+cons latency %v ms, tps %d", cur_epoch, tmpbatchsize, tmpt2-tmpt1, int64(quorum.QuorumSize()*tmpbatchsize*1000)/(tmpt2-tmpt1))
	p = fmt.Sprintf("[Total] %v %v %v %d",cur_epoch, tmpbatchsize, tmpt2-tmpt1, int64(quorum.QuorumSize()*tmpbatchsize*1000)/(tmpt2-tmpt1))
	logging.PrintLog(true,logging.EvaluationLog,p)

}

func ExitTransmissionEpoch(cur_epoch int) {
	if cur_epoch <= 1 {
		return
	}

	tmpt1,_ := pando_t1.Get(cur_epoch)
	tmptrans_t2,_ := trans_t2.Get(cur_epoch)
	tmpbatchsize,_ := batchSize.Get(cur_epoch)

	if tmptrans_t2-tmpt1 == 0 || tmptrans_t2 == 0 || tmpt1 == 0 {
		return
	}

	log.Printf("[Transmission] epoch %v ends with batchsize %v, Transmission latency %v ms, tps %d", cur_epoch, tmpbatchsize, tmptrans_t2-tmpt1, int64(quorum.QuorumSize()*tmpbatchsize*1000)/(tmptrans_t2-tmpt1))
	p := fmt.Sprintf("[Transmission] %v %v %v %d",cur_epoch, tmpbatchsize, tmptrans_t2-tmpt1, int64(quorum.QuorumSize()*tmpbatchsize*1000)/(tmptrans_t2-tmpt1))
	logging.PrintLog(true,logging.EvaluationLog,p)
}

func ExitStateEpoch(cur_epoch int){
	tmpstate_t1,_ := state_t1.Get(cur_epoch)
	tmpstate_t2,_ := state_t2.Get(cur_epoch)
	tmpbatchsize,_ := batchSize.Get(cur_epoch)

	if tmpstate_t2-tmpstate_t1 == 0 || tmpstate_t2 == 0 || tmpstate_t1 == 0 {
		return
	}

	log.Printf("[StateTransfer] epoch %v ends with batchsize %v, latency %v ms", cur_epoch, tmpbatchsize, tmpstate_t2-tmpstate_t1)
	p := fmt.Sprintf("[StateTransfer] %v %v %v",cur_epoch, tmpbatchsize, tmpstate_t2-tmpstate_t1)
	logging.PrintLog(true,logging.EvaluationLog,p)

}

