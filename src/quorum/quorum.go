/*
Verify whether a replica/client has received matching messages from sufficient number of replicas
*/
package quorum

import (
	"fmt"
	"pando/src/message"
	"pando/src/utils"	
	"pando/src/logging"
)

var verbose bool

// Used for normal operation
var buffer BUFFER   //prepare certificate. Client uses it as reply checker.
var bufferc BUFFER  //commit certificate. Client API uses it as reply checker.
var cer CERTIFICATE //used for vcbc only. Store the set of signatures

var intbuffer INTBUFFER // view changes

var n int
var f int
var quorum int
var recQuorum int
var squorum int
var half int

// Used for normal operation
var bufferqc BUFFER
var bufferf BUFFER
var buffernv BUFFER

// Used for view change and checkpoints
var cpbuffer INTBUFFER  // checkpoints

var PCer CERTIFICATE //store seq number and the corresponding prepare message
var QC CERTIFICATE

var k_quorum int
var k_squorum int

type Step int32

const (
	PP Step = 0
	CM Step = 1
	VC Step = 2
	CP Step = 3
	QCP Step = 4 
	FN Step = 5
	Pando Step = 6
	NV Step = 7
)

/*
Clear in-memory data for view changes.
*/
func ClearCer() {
	bufferc.Init()
	buffer.Init()
	cer.Init()
	intbuffer.Init(n)

	PCer.Init()
	bufferf.Init()
	buffernv.Init()
	bufferqc.Init()
	QC.Init()
}

//QCP
func Add(id int64, hash string, input []byte, step Step, seq int) {
	switch step {
	case PP:
		buffer.InsertValue(hash, input, id, step, seq)
	case CM:
		bufferc.InsertValue(hash, input, id, step, seq)
	case QCP:
		bufferqc.InsertValue(hash, input, id, step, seq)
	case FN: 
		bufferf.InsertValue(hash, input, id, step, seq)
	case Pando:
		bufferqc.InsertValue(hash, input, id, step, seq)
	case NV:
		buffernv.InsertValue(hash, input, id, step, seq)
	}
}

func PandoAdd(id int64, hash string, input []byte, step Step, seq int, vrf []byte) {
	switch step {
	case Pando:
		bufferqc.InsertValueWithVrf(hash, input, id, step, seq, vrf)
	case PP:
		buffer.InsertValueWithVrf(hash, input, id, step, seq, vrf)
	}
}


func GetBuffercList(key string) []int64 {
	//_, exist := bufferc.Buffer[key]
	s,exist := bufferc.GetValueBykey(key)
	if exist {
		//s := bufferc.Buffer[key]
		return s.SetList()
	} else {
		return []int64{}
	}
}

//QCP
func CheckQuorum(input string, seq int, step Step) bool {
	switch step {
	case PP:
		return buffer.GetLen(input) >= quorum && PCer.Size(seq) >= quorum
	case CM:
		return bufferc.GetLen(input) >= quorum
	case FN:
		return bufferf.GetLen(input) >= quorum
	}

	return false
}

func CheckBSmallQuorum(input string, seq int, step Step) bool {
	switch step {
	case PP:
		return buffer.GetLen(input) >= quorum && PCer.Size(seq) >= squorum
	case CM:
		return bufferc.GetLen(input) >= squorum
	case QCP:
		//log.Println("CheckBSmallQuorum: ",bufferqc.GetLen(input),squorum,QC.Size(seq)) //
		return bufferqc.GetLen(input) >= squorum && QC.Size(seq) >= squorum
	case Pando:
		//log.Println("CheckBSmallQuorum: ",bufferqc.GetLen(input),k_quorum,QC.Size(seq))
		return bufferqc.GetLen(input) >= k_quorum && QC.Size(seq) >= k_quorum		//k_quorum=k-t
	}

	return false
}

func CheckKQuorumSize(input string, seq int, step Step) bool {
	switch step {
	case NV:
		return buffernv.GetLen(input) >= k_quorum
	case PP:
		return buffer.GetLen(input) >= k_quorum && PCer.Size(seq) >= k_quorum
	case CM:
		return bufferc.GetLen(input) >= k_quorum
	case FN:
		return bufferf.GetLen(input) >= k_quorum
	}

	return false
}

func CheckKSQuorumSize(input string, seq int, step Step) bool {
	switch step {
	case CM:
		return bufferc.GetLen(input) >= k_squorum	////k_squorum=t+1
	}

	return false
}

func CheckCurNum(input string, step Step) int {
	switch step {
	case PP:
		return buffer.GetLen(input)
	case CM:
		return bufferc.GetLen(input)
	}
	return 0
}

func CheckEqualQuorum(input string, step Step) bool {
	switch step {
	case PP:
		return buffer.GetLen(input) == quorum
	case CM:
		return bufferc.GetLen(input) == quorum
	}

	return false
}

// func CheckSmallQuorum(input string, step Step) bool {
// 	switch step {
// 	case PP:
// 		return buffer.GetLen(input) >= squorum
// 	case CM:
// 		return bufferc.GetLen(input) >= squorum
// 	}

// 	return false
// }

func CheckOverSmallQuorum(input string) bool {
	return bufferc.GetLen(input) >= squorum
}

func GetBufferLen(input string) int {
	return buffer.GetLen(input)
}

//QCP
func CheckSmallQuorum(input string) bool {
	return buffer.GetLen(input) == squorum
}

//QCP
func CheckBOverSmallQuorum(input string) bool {
	return buffer.GetLen(input) >= squorum
}


func CheckEqualSmallQuorum(input string) bool {
	return bufferc.GetLen(input) == squorum
}

func ClearBuffer(input string, step Step) {
	switch step {
	case PP:
		buffer.Clear(input)
	case CM:
		bufferc.Clear(input)
	case QCP:
		bufferqc.Clear(input)
	case Pando:
		bufferqc.Clear(input)
	case NV:
		buffernv.Clear(input)
	case FN:
		bufferf.Clear(input)		
	}
}

func ClearBufferPC(input string) {
	buffer.Clear(input)
	bufferc.Clear(input)
}

func AddPCer(key int, value message.Cer) {
	PCer.Add(key, value)
}

func FetchPCer() map[int]message.Cer {
	return PCer.Certificate
}

//QCP
func FetchCer(key int) (message.Cer, bool) {
	result, exist := PCer.Certificate[key]
	if !exist {
		var empty message.Cer
		return empty, false
	}
	return result, true
}

//QCP
func FetchCerIDs(key int) ([]int64, bool) {
	result, exist := PCer.Identities[key]
	if !exist {
		var empty []int64
		return empty, false
	}
	return result, true
}

func FetchCerVRFs(key int) (message.Cer, bool) {
	result, exist := PCer.Vrf[key]
	if !exist {
		var empty message.Cer
		return empty, false
	}
	return result, true	
}

func FetchQC(key int) (message.Cer, bool) {
	result, exist := QC.Certificate[key]
	if !exist {
		var empty message.Cer
		return empty, false
	}
	return result, true
}

func FetchQCIDs(key int) ([]int64, bool) {
	result, exist := QC.Identities[key]
	if !exist {
		var empty []int64
		return empty, false
	}
	return result, true
}

func FetchQCVRFs(key int) (message.Cer, bool) {
	result, exist := QC.Vrf[key]
	if !exist {
		var empty message.Cer
		return empty, false
	}
	return result, true	
}

/*
Add value to IntBuffer. Used for view changes and garbage collection
Input

	view: view number (int)
	source: node that sends the message (int64 type)
	content: MessageWithSignature
	step: type of message, VC for view changes, CP for checkpoints
*/
func AddToIntBuffer(view int, source int64, content message.MessageWithSignature, step Step) {
	switch step {
	case VC:
		intbuffer.InsertValue(view, source, content)
	}

}

/*
Check whether a quorum of messages have been received. Used for view changes and garbage collection.
*/
func CheckIntQuorum(input int, step Step) bool {
	switch step {
	case VC:
		return intbuffer.GetLen(input) >= quorum
	}
	return false
}

/*
Get a quorum of VC/Cp messages
*/
func GetVCMsgs(input int, step Step) []message.MessageWithSignature {
	switch step {
	case VC:
		//return intbuffer.V[input]
		return intbuffer.GetV(input)
	}
	var emptyqueue []message.MessageWithSignature
	return emptyqueue
}

func QuorumSize() int {
	return quorum
}

func RecQuorumSize() int {
	return recQuorum
}

func HalfSize() int {
	return half
}

func SQuorumSize() int {
	return squorum
}

func FSize() int {
	return f
}

func NSize() int {
	return n
}

func GetThreshold(num int) int{
	faulty := (num - 1) / 3
	return faulty + 1
}

// Print quorum info
func PrintQuorumSize() {
	p := fmt.Sprintf("n: %d, f: %d, quorum: %d, small quorum: %d", n, f, quorum, squorum)
	utils.PrintMessage(verbose, p)
}

func SetQuorumSizes(num int) {
	n = num
	f = (n - 1) / 3
	quorum = (n + f + 1) / 2
	if (n+f+1)%2 > 0 {
		quorum += 1
	}
	squorum = f + 1
}

func GetHalf(anum int) int{
	t := (anum - 1) / 2
	half = t + 1
	if (anum+1)%2 > 0 {
		half += 1
	}
	return half
}

func GetT(anum int) int{
	t := (anum - 1) / 2
	return t
}

func CheckOverHalf(input string) bool {
	return bufferc.GetLen(input) >= half
}

func CheckLen(input string) int {
	return bufferc.GetLen(input)
}

func CheckHalf(input string) bool {
	return bufferc.GetLen(input) == half
}

func SetBucketQuorumSizes(knum int, tnum int) {
	k_quorum = knum-tnum
	k_squorum = tnum+1
	if verbose {
		p := fmt.Sprintf("[Replica] n: %d, f: %d, k: %d, t: %d, quorum: %d, k-t: %d, t+1: %d", n, (n-1)/3, knum, tnum, quorum, k_quorum, k_squorum)	
		logging.PrintLog(verbose, logging.EvaluationLog, p)
		p = fmt.Sprintf("[Phase] epoch batchsize Latency TPS")
		logging.PrintLog(true, logging.EvaluationLog, p)		
	}
}

func KQuorumSize() int {
	return k_quorum
}

func KSQuorumSize() int {
	return k_squorum
}

func StartQuorum(num int) {

	n = num
	f = (n - 1) / 3
	quorum = (n + f + 1) / 2
	if (n+f+1)%2 > 0 {
		quorum += 1
	}
	squorum = f + 1

	buffer.Init()
	bufferc.Init()
	cer.Init()
	intbuffer.Init(n)
}

func StartPandoQuorum(num int, anum int) {
	verbose = true

	n = num
	f = (n - 1) / 3
	quorum = (n + f + 1) / 2
	if (n+f+1)%2 > 0 {
		quorum += 1
	}
	squorum = f + 1

	t := (anum - 1) / 2
	half = t + 1
	if (anum+1)%2 > 0 {
		half += 1
	}
	p := fmt.Sprintf("[Replica] n: %d, f: %d, quorum: %d, small quorum: %d, t: %d, half: %d", n, f, quorum, squorum, t, half)
	logging.PrintLog(verbose, logging.NormalLog, p)

	bufferc.Init()
	buffer.Init()
	bufferf.Init()
	intbuffer.Init(n)
	// cpbuffer.Init(n)
	// joinMsgs.Init(n)
	//joinContractMsgs.Init(n)
	//contractRootMsg.Init(n)
	PCer.Init()
	QC.Init()
	bufferqc.Init()
	buffernv.Init()
	// catchupMsgs.Init(100)
	// intervalCatchupMsgs.Init(100)
	//catchupContractMsgs.Init(100)
}

// func StartSleepyHotstuffQuorum(num int, fal int, s int, level string) {
// 	n = num
// 	f = fal
// 	switch level {
// 	case "3f+1":
// 		quorum = num - fal
// 	case "3f+s+1":
// 		quorum = num - fal
// 		recQuorum = num - fal - s
// 	case "3f+2s+1":
// 		quorum = num - fal - s
// 		recQuorum = quorum
// 	}

// 	buffer.Init()
// 	bufferc.Init()
// 	cer.Init()
// 	intbuffer.Init(n)
// }
