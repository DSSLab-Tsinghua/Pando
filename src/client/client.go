package client

import (
	"fmt"
	// "github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5"
	"log"
	sender "pando/src/communication/clientsender"
	"pando/src/config"
	logging "pando/src/logging"
	"pando/src/message"
	pb "pando/src/proto/communication"
	"pando/src/utils"
	// "pando/src/cryptolib"
)

var cid int64
var err error
var clientTimer int

func GetCID() int64 {
	return cid
}

// func SignedRequest(cid int64, dataSer []byte) ([]byte, bool) {
// 	request := message.MessageWithSignature{
// 		Msg: dataSer,
// 		Sig: []byte(""), //cryptolib.GenSig(cid, dataSer),
// 	}

// 	requestSer, err := request.Serialize()
// 	if err != nil {
// 		p := fmt.Sprintf("[Client error] fail to serialize the request with signiture: %v", err)
// 		logging.PrintLog(true, logging.ErrorLog, p)
// 		return []byte(""), false
// 	}

// 	return requestSer, true
// }

func SignedRequest(cid int64, dataSer []byte) ([]byte, bool) {
	request := message.MessageWithSignature{
		Msg:     dataSer,
		Sig: 	[]byte(""), //cryptolib.GenSig(cid, dataSer),
	}

	requestSer, err := request.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the request with signiture: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return requestSer, true
}

func CreateAWriteTransaction(wType int, uid string, op string, ac []byte, tnum int, ts int64, txlist []string, shard int64) []byte {
	var dataSer []byte 
	 
	if config.EvalMode() > 0{
		data := message.ClientRequest{
			Type:  pb.MessageType_WRITE,
			ID:    cid,
			TS:    ts, 
		}
	
		dataSer, _ = data.Serialize()
	}else{
		log.Printf("pass here")
		// data1, result := CreateWriteRequest(wType, uid, utils.StringToBytes(op), "", "", txlist)
		// if !result {
		// 	log.Fatalf("Failed to create write request")
		// 	return []byte("")
		// }
	
		// dataSer, result = CreateRequestFromExternalTS(cid, data1, ac, tnum, ts, shard)
		// if !result {
		// 	log.Fatalf("Failed to create transaction")
		// 	return []byte("")
		// }
	}
	
	requestSer, result2 := SignedRequest(cid, dataSer)
	if !result2 {
		log.Fatalf("Failed to sign the message")
		return []byte("")
	}
	return requestSer
}


func SendWriteRequest(wType int, uid string, op string, ac []byte, tnum int, txlist []string, shard int64) {

	// dataSer, result1 := CreateRequest(cid, op)
	// if !result1 {
	// 	return
	// }

	// requestSer, result2 := SignedRequest(cid, dataSer)
	// if !result2 {
	// 	return
	// }
	// log.Println("len of request: ", len(requestSer))
	// sender.BroadcastRequest(pb.MessageType_WRITE, requestSer)

	ts := utils.MakeTimestamp()
	requestSer := CreateAWriteTransaction(wType, uid, op, ac, tnum, ts, txlist, shard)

	sender.BroadcastRequest(pb.MessageType_WRITE, requestSer)
	if config.EvalMode() > 0 {
		return
	}	
}

func SendBatchRequests(wType int, uid string, op string, batch int, ac []byte, tnum int) {

	var requests [][]byte
	ts := utils.MakeTimestamp()
	data := CreateAWriteTransaction(wType, uid, op, ac, tnum, ts, nil, 0)
	//log.Printf("###Client default request len is %v", len(data))
	for i := 0; i < batch; i++ {
		/*rawdata := message.ClientRequest{
			ID: cid,
			OP: utils.StringToBytes(op),
			TS: utils.MakeTimestamp(),
		}
		data,_ := rawdata.Serialize()*/
		requests = append(requests, data)
	}

	dataSer, _ := SerializeRequests(requests)
	log.Printf("###Client default request len is %v", len(dataSer))
	//quorum.ClearCer()

	sender.BroadcastRequest(pb.MessageType_WRITE_BATCH, dataSer)
}


func SendReconstructRequest(instance int) {

	op := utils.IntToBytes(instance)
	dataSer, result1 := CreateReconstructRequest(cid, op)
	if !result1 {
		return
	}

	requestSer, result2 := SignedRequest(cid, dataSer)
	if !result2 {
		return
	}
	log.Println("len of request: ", len(requestSer))
	sender.BroadcastRequest(pb.MessageType_RECONSTRUCT, requestSer)
}

func SendTestHacssRequest() {

	dataSer, result1 := CreateTestHacssRequest(cid)
	if !result1 {
		return
	}

	requestSer, result2 := SignedRequest(cid, dataSer)
	if !result2 {
		return
	}
	log.Println("len of request: ", len(requestSer))
	sender.BroadcastRequest(pb.MessageType_TEST_HACSS, requestSer)
}

func SendBatchRequest(op []byte, bitchSize int) {
	var requestArr [][]byte
	for i := 0; i < bitchSize; i++ {

		dataSer, result1 := CreateRequest(cid, op)
		if !result1 {
			return
		}

		requestSer, result2 := SignedRequest(cid, dataSer)
		if !result2 {
			return
		}
		//log.Println("len of request in batch: ",len(requestSer))
		requestArr = append(requestArr, requestSer)
	}
	byteRequsets, err := SerializeRequests(requestArr)
	if err != nil {
		log.Fatal("[Client error] fail to serialize the message.")
	}

	sender.BroadcastRequest(pb.MessageType_WRITE_BATCH, byteRequsets)
}

func CreateRequest(cid int64, op []byte) ([]byte, bool) {
	data := message.ClientRequest{
		Type: pb.MessageType_WRITE,
		ID:   cid,
		OP:   op,
		TS:   utils.MakeTimestamp(),
	}

	dataSer, err := data.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the write request: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return dataSer, true
}

func CreateReconstructRequest(cid int64, instance []byte) ([]byte, bool) {
	data := message.ClientRequest{
		Type: pb.MessageType_RECONSTRUCT,
		ID:   cid,
		OP:   instance,
		TS:   utils.MakeTimestamp(),
	}

	dataSer, err := data.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the write request: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return dataSer, true
}

func CreateTestHacssRequest(cid int64) ([]byte, bool) {
	data := message.ClientRequest{
		Type: pb.MessageType_TEST_HACSS,
		ID:   cid,
		TS:   utils.MakeTimestamp(),
	}

	dataSer, err := data.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the write request: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return dataSer, true
}

/*
Serialize data into a json object in bytes
Output

	[]byte: serialized request
	error: err is nil if request is serialized
*/
func SerializeRequests(r [][]byte) ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the message %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), err
	}
	return jsons, nil
}

func StartClient(rid string, loadkey bool) {
	logging.SetID(rid)
	config.LoadConfig()
	logging.SetLogOpt(config.FetchLogOpt())

	log.Printf("Client %s started.", rid)
	cid, err = utils.StringToInt64(rid)
	sender.StartClientSender(rid, loadkey)
	clientTimer = config.FetchBroadcastTimer()
}
