package consensus

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	// "time"
	"math"
	"crypto/sha256"
	"math/big"

	pb "pando/src/proto/communication"
	message "pando/src/message"
	"pando/src/quorum"
	"pando/src/utils"
	sequence "pando/src/consensus/sequence"

	sender "pando/src/communication/sender"
	// "pando/src/communication"
	"pando/src/logging"
	"pando/src/config"
	// "pando/src/cryptolib"
	"github.com/cbergoon/merkletree"
	bucket "pando/src/consensus/bucket"
	vrf "github.com/yoseplee/vrf"
	"github.com/klauspost/reedsolomon"
	prf "pando/src/cryptolib/threshprf"
)

var receivedFrag utils.IntBytesMap
var decodedInstance utils.IntBytesMap //decode the erasure for instance upon receive f+1 frags
var decodeStatus utils.Set            //set true if decode
var entireInstance utils.IntByteMap   //set the decoded instance payload

var publicKeyMap utils.IntByteMap
var privateKeyMap utils.IntByteMap

var cTransProofMap utils.IntByteMap //map[epoch]pi
var c1ConsProofMap utils.IntByteMap
var c2ConsProofMap utils.IntByteMap
var c3ConsProofMap utils.IntByteMap
var c1StateProofMap utils.IntByteMap
var c2StateProofMap utils.IntByteMap

var cTransHashMap utils.IntByteMap //map[epoch]hash
var c1ConsHashMap utils.IntByteMap
var c2ConsHashMap utils.IntByteMap
var c3ConsHashMap utils.IntByteMap
var c1StateHashMap utils.IntByteMap
var c2StateHashMap utils.IntByteMap

var cTransIsCommitteeMap utils.IntBoolMap //map[epoch]true
var c1ConsIsCommitteeMap utils.IntBoolMap
var c2ConsIsCommitteeMap utils.IntBoolMap
var c3ConsIsCommitteeMap utils.IntBoolMap
var c1StateIsCommitteeMap utils.IntBoolMap
var c2StateIsCommitteeMap utils.IntBoolMap

var cIsBucketMap utils.IntIntMap
var retrievalStatus utils.IntBoolMap
var awaitingFragment utils.IntByteMap
var retrievalEnd utils.IntBoolMap

var consensusGlobalStatus utils.IntBoolMap
var stateTransferHash utils.IntByteMap
var qcplock sync.Mutex

var bufferLock sync.Mutex
var buffer utils.StringIntMap


func SetBucketConfig() {
	totalReplicas := config.FetchNumReplicas()

	k := math.Floor(config.CommitteeSizes() * float64(totalReplicas))
	t := math.Floor((k - 1) / 3)

	bucket.UpdateKBucketNum(int(k))
	bucket.UpdateTBucketFailure(int(t))
}

func SetBucketsForTransfer() {
	totalReplicas := config.FetchNumReplicas()
	var b int
	b = 1

	for index := 0; index < totalReplicas; index++ {
		if b > bucket.GetKBucketNum() {
			b = 1
		}
		cIsBucketMap.Insert(index, b)
		b++
	}
}

func CommitteeSetUp(epochHardcode int) {
	totalReplicas := config.FetchNumReplicas()
	SetKeyPairs()

	// ComProveStart := utils.MakeTimestamp()
	for e := 0; e < epochHardcode; e++ {
		for i := 0; i < totalReplicas; i++ {
			index := (totalReplicas * e) + i

			commsg := message.ComMessage{
				Seq:     e + 1,
				Process: "transmission",
				Source:  int64(i),
			}
			commsgbyte, _ := commsg.Serialize()
			pi, hash, _ := GenVrfProof(commsgbyte, int64(i))
			cTransProofMap.Insert(index, pi)
			//temp_hash := vrf.Hash(pi) //temp_hash equals to hash
			//log.Printf("\nif pass ComProve: %v", ComProve(pi,commsgbyte,int64(i))) //false, why???
			cTransHashMap.Insert(index, hash)
			cTransIsCommitteeMap.Insert(index, ComVerify(hash, int64(index)))

			commsg = message.ComMessage{
				Seq:     e + 1,
				Process: "consensus1",
				Source:  int64(i),
			}
			commsgbyte, _ = commsg.Serialize()
			pi, hash, _ = GenVrfProof(commsgbyte, int64(i))
			c1ConsProofMap.Insert(index, pi)
			c1ConsHashMap.Insert(index, hash)
			c1ConsIsCommitteeMap.Insert(index, ComVerify(hash, int64(index)))

			commsg = message.ComMessage{
				Seq:     e + 1,
				Process: "consensus2",
				Source:  int64(i),
			}
			commsgbyte, _ = commsg.Serialize()
			pi, hash, _ = GenVrfProof(commsgbyte, int64(i))
			c2ConsProofMap.Insert(index, pi)
			c2ConsHashMap.Insert(index, hash)
			c2ConsIsCommitteeMap.Insert(index, ComVerify(hash, int64(index)))

			commsg = message.ComMessage{
				Seq:     e + 1,
				Process: "consensus3",
				Source:  int64(i),
			}
			commsgbyte, _ = commsg.Serialize()
			pi, hash, _ = GenVrfProof(commsgbyte, int64(i))
			c3ConsProofMap.Insert(index, pi)
			c3ConsHashMap.Insert(index, hash)
			c3ConsIsCommitteeMap.Insert(index, ComVerify(hash, int64(index)))

			commsg = message.ComMessage{
				Seq:     e + 1,
				Process: "transfer1",
				Source:  int64(i),
			}
			commsgbyte, _ = commsg.Serialize()
			pi, hash, _ = GenVrfProof(commsgbyte, int64(i))
			c1StateProofMap.Insert(index, pi)
			c1StateHashMap.Insert(index, hash)
			c1StateIsCommitteeMap.Insert(index, ComVerify(hash, int64(index)))

			commsg = message.ComMessage{
				Seq:     e + 1,
				Process: "transfer2",
				Source:  int64(i),
			}
			commsgbyte, _ = commsg.Serialize()
			pi, hash, _ = GenVrfProof(commsgbyte, int64(i))
			c2StateProofMap.Insert(index, pi)
			c2StateHashMap.Insert(index, hash)
			c2StateIsCommitteeMap.Insert(index, ComVerify(hash, int64(index)))

		}
	}
	// ComProveStop := utils.MakeTimestamp()
	// log.Printf("&&&&&&&&&&&&&&&&ComProve processed using %v ms.", ComProveStop-ComProveStart)

}

func VerifyCommittee(epochHardcode int) {
	totalReplicas := config.FetchNumReplicas()

	for e := 0; e < epochHardcode; e++ {
		ctestlist := make([]bool, totalReplicas)
		c1testlist := make([]bool, totalReplicas)
		c2testlist := make([]bool, totalReplicas)
		c3testlist := make([]bool, totalReplicas)
		c11testlist := make([]bool, totalReplicas)
		c22testlist := make([]bool, totalReplicas)
		ctrue := 0
		c1true := 0
		c2true := 0
		c3true := 0
		c11true := 0
		c22true := 0

		for i := 0; i < totalReplicas; i++ {
			index := (totalReplicas * e) + i

			ctestlist[i], _ = cTransIsCommitteeMap.Get(index)
			if ctestlist[i] == true {
				ctrue = ctrue + 1
			}

			c1testlist[i], _ = c1ConsIsCommitteeMap.Get(index)
			if c1testlist[i] == true {
				c1true = c1true + 1
			}

			c2testlist[i], _ = c2ConsIsCommitteeMap.Get(index)
			if c2testlist[i] == true {
				c2true = c2true + 1
			}

			c3testlist[i], _ = c3ConsIsCommitteeMap.Get(index)
			if c3testlist[i] == true {
				c3true = c3true + 1
			}

			c11testlist[i], _ = c1StateIsCommitteeMap.Get(index)
			if c11testlist[i] == true {
				c11true = c11true + 1
			}

			c22testlist[i], _ = c2StateIsCommitteeMap.Get(index)
			if c22testlist[i] == true {
				c22true = c22true + 1
			}
		}

		if ctrue < bucket.GetKBucketNum() {
			p := fmt.Sprintf("[VerifyCommittee] Members in cTrans committee with len %v not verified, for epoch %v. ", ctrue, e+1)
			logging.PrintLog(verbose, logging.NormalLog, p)
		}

		if c1true < bucket.GetKBucketNum() {
			p := fmt.Sprintf("[VerifyCommittee] Members in c1Cons committee with len %v not verified, for epoch %v. ", c1true, e+1)
			logging.PrintLog(verbose, logging.NormalLog, p)
		}

		if c2true < bucket.GetKBucketNum() {
			p := fmt.Sprintf("[VerifyCommittee] Members in c2Cons committee with len %v not verified, for epoch %v. ", c2true, e+1)
			logging.PrintLog(verbose, logging.NormalLog, p)
		}

		if c3true < bucket.GetKBucketNum() {
			p := fmt.Sprintf("[VerifyCommittee] Members in c3Cons committee with len %v not verified, for epoch %v. ", c3true, e+1)
			logging.PrintLog(verbose, logging.NormalLog, p)
		}

		if c11true < bucket.GetKBucketNum() {
			p := fmt.Sprintf("[VerifyCommittee] Members in c1State committee with len %v not verified, for epoch %v. ", c11true, e+1)
			logging.PrintLog(verbose, logging.NormalLog, p)
		}

		if c22true < bucket.GetKBucketNum() {
			p := fmt.Sprintf("[VerifyCommittee] Members in c2State committee with len %v not verified, for epoch %v. ", c22true, e+1)
			logging.PrintLog(verbose, logging.NormalLog, p)
		}

	}

}

func SetKeyPairs() {
	totalReplicas := config.FetchNumReplicas()
	for i := 0; i < totalReplicas; i++ {
		privateKey, _, _ := prf.LoadkeyFromFiles(int64(i))
		// privateKey, publicKey := smgo.GB_LoadKeysFromFile(int64(i))
		publicKey := prf.PandoLoadvkFromFiles(int64(i))

		privateKeyMap.Insert(i, privateKey)
		publicKeyMap.Insert(i, publicKey)
	}

}

func GenVrfProof(message []byte, id int64) ([]byte, []byte, error) {
	privateKey, _ := privateKeyMap.Get(int(id))
	publicKey, _ := publicKeyMap.Get(int(id))
	pi, hash, err := vrf.Prove(publicKey, privateKey, message[:])
	if err != nil {
		log.Println("Fail to generate GenVrfProof: ", err)
		return nil, nil, err
	}

	return pi, hash, nil
}

func ComVerify(hash []byte, jid int64) bool {

	ratio := HashRatio(hash)

	res := Sortition(ratio) //if ratio < sortitionThreshold, return true (in paper, sortitionThreshold is the difficulty parameter D)
	if !res {
		//log.Println("[ComProve] Source node is not in the committee.")
		// p := fmt.Sprintf("[ComProve] My ratio is %v, I'm not in the committee(%v). ", ratio, jid)
		// logging.PrintLog(verbose, logging.NormalLog, p)
		return false
	} else {
		// p := fmt.Sprintf("[ComProve] My ratio is %v, I'm in the committee(%v). ", ratio, jid)
		// logging.PrintLog(verbose, logging.NormalLog, p)

		return true
	}

}

func ComProve(vrfPi []byte, message []byte, index int, process string) bool {
	var source_hash []byte
	var target_hash []byte

	if process == "transmission" {
		source_hash = vrf.Hash(vrfPi)
		target_hash, _ = cTransHashMap.Get(index)
	} else if process == "consensus1" {
		source_hash = vrf.Hash(vrfPi)
		target_hash, _ = c1ConsHashMap.Get(index)
	} else if process == "consensus2" {
		source_hash = vrf.Hash(vrfPi)
		target_hash, _ = c2ConsHashMap.Get(index)
	} else if process == "consensus3" {
		source_hash = vrf.Hash(vrfPi)
		target_hash, _ = c3ConsHashMap.Get(index)
	} else if process == "transfer1" {
		source_hash = vrf.Hash(vrfPi)
		target_hash, _ = c1StateHashMap.Get(index)
	} else if process == "transfer2" {
		source_hash = vrf.Hash(vrfPi)
		target_hash, _ = c2StateHashMap.Get(index)
	}

	return bytes.Equal(source_hash, target_hash)
}

/*
The function will encode input to erasure code. input is the data that to be encoded; dataShards is the minimize number that decoding;
totalShards is the total number that encoding
*/
func ErasureEncoding(input []byte, dataShards int, totalShards int) ([][]byte, bool) {
	// var ec_encode_t1 int64
	// var ec_encode_t2 int64

	// ec_encode_t1 = utils.MakeTimestamp()
	if dataShards == 0 {
		return [][]byte{}, false
	}
	//log.Println("len of input: ",len(input),input)
	enc, err := reedsolomon.New(dataShards, totalShards-dataShards)
	if err != nil {
		log.Println("Fail to execute New() in reed-solomon: ", err)
		return [][]byte{}, false
	}

	PaddingInput(&input, dataShards)
	//log.Println("len of input: ",len(input),input)

	data := make([][]byte, totalShards)
	paritySize := len(input) / dataShards
	//log.Println("paritySize: ",paritySize)

	for i := 0; i < totalShards; i++ {
		data[i] = make([]byte, paritySize)
		if i < dataShards {
			data[i] = input[i*paritySize : (i+1)*paritySize]
		}

	}
	//log.Println("len of data: ",len(data),data)
	err = enc.Encode(data)
	if err != nil {
		log.Println("Fail to encode the input to erasure code: ", err)
		return nil, false
	}
	ok, err1 := enc.Verify(data)
	if err1 != nil || !ok {
		log.Println("Fail verify the erasure code: ", err)
		return nil, false
	}
	// ec_encode_t2 = utils.MakeTimestamp()
	// log.Println("Encoding Latency = %v ms.", ec_encode_t2 - ec_encode_t1)

	//log.Println("len of data: ",len(data),data)
	return data, true
}

func ErasureDecoding(instanceID int, dataShards int, totalShards int) {
	// var ec_decode_t1 int64
	// var ec_decode_t2 int64

	// ec_decode_t1 = utils.MakeTimestamp()
	if decodeStatus.HasItem(int64(instanceID)) {
		return
	}
	ids, frags := receivedFrag.GetAllValue(instanceID)
	data := make([][]byte, totalShards)

	for index, ID := range ids {
		data[ID] = frags[index]
	}

	entireIns := DecodeData(data, dataShards, totalShards)
	decodeStatus.AddItem(int64(instanceID))
	decodedInstance.SetValue(instanceID, data)
	//log.Printf("[%v] Decode the erasure code to : %v",instanceID,entireIns)
	entireInstance.Insert(instanceID, entireIns)

	// ec_decode_t2 = utils.MakeTimestamp()
	// log.Println("Decoding Latency = %v ms.", ec_decode_t2-ec_decode_t1)
}

/*
if the length of input is not an integer multiple of size, padding "0" in the end
*/
func PaddingInput(input *[]byte, size int) {
	if size == 0 {
		return
	}
	initLen := len(*input)
	remainder := initLen % size
	if remainder == 0 {
		return
	} else {
		ending := make([]byte, remainder)
		*input = append(*input, ending[:]...)
	}
}

func DecodeData(data [][]byte, dataShards int, totalShards int) []byte {
	enc, err := reedsolomon.New(dataShards, totalShards-dataShards)
	if err != nil {
		log.Println("Fail to execute New() in reed-solomon: ", err)
		return nil
	}
	err = enc.Reconstruct(data)
	if err != nil {
		log.Println("Fail to decode the erasure conde: ", err)
		return nil
	}
	//log.Printf("*******Decode erasure: %s",data)

	var entireIns []byte

	for i := 0; i < bucket.GetTBucketFailure()+1; i++ {
		entireIns = append(entireIns, data[i]...)
	}

	// for i:=0;i<totalShards-dataShards;i++{
	// 	entireIns = append(entireIns, data[i]...)
	// }

	for {
		if entireIns == nil || len(entireIns) == 0 {
			return nil
		}
		if entireIns[len(entireIns)-1] == 0 {
			entireIns = entireIns[0 : len(entireIns)-1]
		} else {
			break
		}
	}
	return entireIns
}

func Sortition(ratio float64) bool {
	// log.Println("ratio caculated: ", ratio)
	if ratio > config.SortitionThreshold() {	//if ratio < sortitionThreshold, return true (in paper, sortitionThreshold is the difficulty parameter D)
		return false
	}
	return true
}

// HashRatio calculates a float number between [0, 1] with a random hash value which generated by vrf
func HashRatio(vrfOutput []byte) float64 {

	t := &big.Int{}
	t.SetBytes(vrfOutput[:])

	precision := uint(8 * (len(vrfOutput) + 1))
	max, b, err := big.ParseFloat("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0, precision, big.ToNearestEven)
	if b != 16 || err != nil {
		log.Fatal("failed to parse big float constant for sortition")
	}

	//hash value as int expression.
	//hval, _ := h.Float64() to get the value
	h := big.Float{}
	h.SetPrec(precision)
	h.SetInt(t)

	ratio := big.Float{}
	cratio, _ := ratio.Quo(&h, max).Float64()

	return cratio
}

type MTContent struct {
	x []byte
}

func (t MTContent) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(t.x)); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func (t MTContent) Equals(other merkletree.Content) (bool, error) {
	return bytes.Compare(t.x, other.(MTContent).x) == 0, nil
}

func ObtainMerkleNodeHash(input []byte) []byte {
	mn := MTContent{x: input}
	h, _ := mn.CalculateHash()
	return h
}

func GenMerkleTreeRoot(input [][]byte) []byte {
	var list []merkletree.Content
	for i := 0; i < len(input); i++ {
		list = append(list, MTContent{x: input[i]})
	}
	t, err := merkletree.NewTree(list)
	if err != nil {
		return nil
	}
	mr := t.MerkleRoot()
	return mr
}

func ObtainMerklePath(input [][]byte) ([][][]byte, [][]int64) {

	var result [][][]byte
	var indexresult [][]int64

	var list []merkletree.Content
	for i := 0; i < len(input); i++ {
		list = append(list, MTContent{x: input[i]})
	}
	t, err := merkletree.NewTree(list)
	if err != nil {
		return nil, nil
	}

	for idx := 0; idx < len(input); idx++ {
		if idx > len(t.Leafs) {
			return nil, nil
		}
		curNode := t.Leafs[idx]
		intermediate, index, _ := t.GetMerklePath(curNode.C)
		result = append(result, intermediate)
		indexresult = append(indexresult, index)
	}

	return result, indexresult
}

func VerifyMerkleRoot(root []byte, rd []byte, branch [][]byte, index []int64) bool {
	hash := ObtainMerkleNodeHash(rd)
	for i := 0; i < len(index); i++ {
		if index[i]%2 == 0 { //leftnode
			chash := append(branch[i], hash...)
			hash = ObtainMerkleNodeHash(chash)
		} else {
			chash := append(hash, branch[i]...)
			hash = ObtainMerkleNodeHash(chash)
		}
	}

	return bytes.Compare(hash, root) == 0
}

func HandleQCPNewView(content message.ReplicaMessage) {
	dstatus, _ := deliverStatus.Get(content.Seq)
	cstatus, _ := commitStatus.Get(content.Seq)
	if dstatus || cstatus {
		return
	}

	commsg := message.ComMessage{
		Seq:     content.Seq,
		Process: "consensus1",
		Source:  content.Source,
	}
	commsgbyte, _ := commsg.Serialize()
	totalReplicas := config.FetchNumReplicas()
	index := (totalReplicas * (content.Seq - 1)) + int(content.Source)

	if !ComProve(content.VrfPi, commsgbyte, index, "consensus1") {
		p := fmt.Sprintf("[Consensus1] vrfPi from node %v at seq %v not verified", commsg.Source, content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	if content.PreHash != nil {
		blockinfo := message.DeserializeQCBlock(content.PreHash)
		//lockQC的高度是否比现在qchigh高，验签lockQC
		if blockinfo.Height > curBlock.Height {
			if !VerifyBlock(blockinfo.Height, blockinfo) {
				p := fmt.Sprintf("[QC] LockQC Block %d not verified, proposed block height %v, seq %v", blockinfo.Height, curBlock.Height, content.Seq)
				logging.PrintLog(true, logging.ErrorLog, p)
				return
			}
			// if !VerifyVrf(blockinfo, "consensus2") {
			// 	log.Printf("Failed to VRF in seq %v, consensus2 qcblock seq is %v.", content.Seq, qcblockinfo.Height)
			// 	return
			// }

			curBlock = blockinfo
			log.Printf("Store the QChigh for %v.", content.Seq)
		}
	}

	// hash := utils.BytesToString(content.Hash)
	// quorum.Add(content.Source, hash, content.PreHash, quorum.NV, content.Seq)
	NVcounter.Increment(content.Seq)
	tmpNum, _ := NVcounter.Get(content.Seq)

	status, _ := consensusGlobalStatus.Get(content.Seq)
	//if quorum.CheckKQuorumSize(hash, content.Seq, quorum.NV) && !status {	//k-t
	if tmpNum >= quorum.KQuorumSize() && !status {
		consensusGlobalStatus.Insert(content.Seq, true)

		log.Printf("[QC] Processing LockQC proposed at height %v", content.Seq) //cannot print out when batchsize is large
		if !CheckLeader(content.Seq, id) {
			if verbose {
				log.Printf("[Replica] I'm not a leader in seq %v, stop propose in consensus process.", content.Seq)
				logging.PrintLog(verbose, logging.NormalLog, "[Replica] I'm not a leader, stop propose in consensus process.")
			}
			return
		}

		curStatusSeq = content.Seq
		curConsensusStatus.Set(READY)

	}
	
}

func HandleDistribute(content message.ReplicaMessage) {
	commsg := message.ComMessage{
		Seq:     content.Seq,
		Process: "transfer1",
		Source:  content.Source,
	}
	commsgbyte, _ := commsg.Serialize()
	totalReplicas := config.FetchNumReplicas()
	index := (totalReplicas * (content.Seq - 1)) + int(content.Source)

	if !ComProve(content.VrfPi, commsgbyte, index, "transfer1") {
		p := fmt.Sprintf("[Transfer1] vrfPi from node %v at epoch %v not verified", commsg.Source, content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	var tempres bool
	index = (totalReplicas * (content.Seq - 1)) + int(id)
	tempres, _ = c2StateIsCommitteeMap.Get(index)
	VrfPi, _ := c2StateProofMap.Get(index)

	stStatus, _ := retrievalEnd.Get(content.Seq)
	if tempres && !stStatus {
		temp, _ := cIsBucketMap.Get(int(id))
		if temp != content.Num {
			p := fmt.Sprintf("[Transfer1] I'm in bucket %v, received distrubute msg should send to bucket %v.", temp, content.Num)
			logging.PrintLog(true, logging.ErrorLog, p)
			return
		}

		res := VerifyMerkleRoot(content.TransactionRoot, content.Fragment, content.Witness, content.Index)
		if res {
			msg := message.ReplicaMessage{
				Mtype:           pb.MessageType_QCREP, //go HandleShare
				Seq:             content.Seq,
				Source:          id,
				Num:             temp,
				TransactionRoot: content.TransactionRoot,
				Witness:         content.Witness,
				Fragment:        content.Fragment,
				Index:           content.Index,
				VrfPi:           VrfPi,
			}

			msgbyte, err := msg.Serialize()
			if err != nil {
				logging.PrintLog(true, logging.ErrorLog, "[Replica Error] Not able to serialize the message.")
			} else {
				//log.Printf("Broadcast the Share message to all")
				logging.PrintLog(verbose, logging.NormalLog, "[Replica] Broadcast the Share message to all")
				request, _ := message.SerializeWithSignature(id, msgbyte)
				HandleQCByteMsg(request)

				sender.RBCByteBroadcast(msgbyte)
			}

		} else {
			log.Fatal("[VerifyMerkleRoot] fail to verify merkle root during state transfer!")
		}
	}

}

func HandleShare(content message.ReplicaMessage) {
	tmpt, _ := state_t1.Get(content.Seq)
	if tmpt == 0 {
		state_t1.Insert(content.Seq, utils.MakeTimestamp())
	}

	temp, _ := cIsBucketMap.Get(int(content.Source))
	if temp != content.Num {
		p := fmt.Sprintf("[Transfer2] Received distrubute msg not from the correct bucket.")
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	commsg := message.ComMessage{
		Seq:     content.Seq,
		Process: "transfer2",
		Source:  content.Source,
	}
	commsgbyte, _ := commsg.Serialize()
	totalReplicas := config.FetchNumReplicas()
	index := (totalReplicas * (content.Seq - 1)) + int(content.Source)

	if !ComProve(content.VrfPi, commsgbyte, index, "transfer2") {
		p := fmt.Sprintf("[Transfer2] vrfPi from node %v at epoch %v not verified", commsg.Source, content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	res := VerifyMerkleRoot(content.TransactionRoot, content.Fragment, content.Witness, content.Index)
	if res {
		_, exist := awaitingFragment.Get(content.Num - 1)
		if !exist {
			log.Printf("[HandleShare] collecting frags for bucket index %v.", content.Num-1)
			awaitingFragment.Insert(content.Num-1, content.Fragment)
		}

		status, _ := retrievalStatus.Get(int(content.Source))
		stStatus, _ := retrievalEnd.Get(content.Seq)
		if awaitingFragment.GetLen() >= bucket.GetTBucketFailure()+1 && !status && !stStatus {
			frags := make([][]byte, bucket.GetKBucketNum())
			for k, v := range awaitingFragment.GetAll() {
				frags[k] = v
			}

			originalVal := DecodeData(frags, bucket.GetTBucketFailure()+1, bucket.GetKBucketNum())
			if originalVal == nil {
				log.Printf("[HandleShare] originalVal nil, continue collecting fragments.")
				return
			}

			testData, suc := ErasureEncoding(originalVal, bucket.GetTBucketFailure()+1, bucket.GetKBucketNum())
			if !suc {
				log.Printf("[HandleShare] Fail to apply erasure coding during state transfer!")
				return
			}
			testRoot := GenMerkleTreeRoot(testData)

			if bytes.Compare(testRoot, content.TransactionRoot) == 0 {
				retrievalStatus.Insert(int(content.Source), true)
				retrievalEnd.Insert(content.Seq, true)

				log.Printf("[HandleShare] retrieval success...")
				awaitingFragment.Init()

				curStateTransStatus.Set(READY)

				state_t2.Insert(content.Seq, utils.MakeTimestamp())
				ExitStateEpoch(content.Seq)

			} else {
				log.Printf("[HandleShare] root compare fail, continue collecting fragments.")
			}

		}

	} else {
		log.Fatal("[VerifyMerkleRoot] fail to verify merkle root during state transfer!")
	}

}

func InitPando() {

	buffer.Init()
	bcStatus.Init()
	cachedViewMsg.Init()

	timeoutBuffer.Init(n)
	bucket.InitBucket()
	sequence.InitSequence()
	
	SetBucketConfig()
	quorum.SetBucketQuorumSizes(bucket.GetKBucketNum(), bucket.GetTBucketFailure())

	prf.SetHomeDir()
	publicKeyMap.Init()
	privateKeyMap.Init()

	cTransProofMap.Init()
	c1ConsProofMap.Init()
	c2ConsProofMap.Init()
	c3ConsProofMap.Init()
	c1StateProofMap.Init()
	c2StateProofMap.Init()

	cTransHashMap.Init()
	c1ConsHashMap.Init()
	c2ConsHashMap.Init()
	c3ConsHashMap.Init()
	c1StateHashMap.Init()
	c2StateHashMap.Init()

	cTransIsCommitteeMap.Init()
	c1ConsIsCommitteeMap.Init()
	c2ConsIsCommitteeMap.Init()
	c3ConsIsCommitteeMap.Init()
	c1StateIsCommitteeMap.Init()
	c2StateIsCommitteeMap.Init()

	CommitteeSetUp(5) //hardcoded, prepare all vrf for 5 epoches
	VerifyCommittee(5)

	cIsBucketMap.Init()
	SetBucketsForTransfer()

	//log.Printf("\ncTransIsCommitteeMap %v\nc1ConsIsCommitteeMap %v\nc2ConsIsCommitteeMap %v\nc3ConsIsCommitteeMap %v", cTransIsCommitteeMap, c1ConsIsCommitteeMap, c2ConsIsCommitteeMap, c3ConsIsCommitteeMap)
	p := fmt.Sprintf("\ncTransIsCommitteeMap %v\nc1ConsIsCommitteeMap %v\nc2ConsIsCommitteeMap %v\nc3ConsIsCommitteeMap %v\nc1StateIsCommitteeMap %v\nc2StateIsCommitteeMap %v\ncIsBucketMap %v", cTransIsCommitteeMap, c1ConsIsCommitteeMap, c2ConsIsCommitteeMap, c3ConsIsCommitteeMap, c1StateIsCommitteeMap, c2StateIsCommitteeMap, cIsBucketMap)
	logging.PrintLog(verbose, logging.NormalLog, p)

	TransmissionStatus.Init()
	curTransmissionQC.Init() //qc
	epochOPSMap.Init()       //proposal

	NVcounter.Init()
	curNewViewStatus.Init()
	curConsensusStatus.InitWithBlock()
	curStateTransStatus.Init()
	stateTransferHash.Init()
	retrievalStatus.Init()
	awaitingFragment.Init()
	retrievalEnd.Init()

	//for evaluation
	batchSize.Init()
	cons_t1.Init()
	trans_t2.Init()
	pando_t1.Init()
	pando_t2.Init()
	state_t1.Init()
	state_t2.Init()

}
