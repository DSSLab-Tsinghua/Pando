package sequence

import (
	"log"
)


func GetEpoch() int {
	return Epoch.Get()
}

func EpochIncrement() int {
	return Epoch.Increment()
}

func GetLEpoch() int {
	return LEpoch.Get()
}

//QCP
func EpochLIncrement() int {
	return LEpoch.Increment()
}

func UpdateLEpoch(seq int) {
	if seq > GetLEpoch() {
		LEpoch.Set(seq)
	}
}


func GetSeq() int {
	return Sequence.Get()
}

func GetLSeq() int {
	return Lsequence.Get()
}

func Increment() int {
	return Sequence.Increment()
}

//QCP
func IncrementLSeq() int {
	return Lsequence.Increment()
}

func UpdateLSeq(seq int) {
	if seq > GetLSeq() {
		Lsequence.Set(seq)
	}
}

func UpdateSeq(seq int) {
	if seq > GetSeq() {
		Sequence.Set(seq)
	}
}

func ResetLSeq(seq int) {
	Lsequence.Set(seq)
}

func ResetSeq(seq int) {
	Sequence.Set(seq)
}


/* sharding mode */
func ShardGetSeq(key int64) int {
	v,_ := Shardseqs.Get(key)
	return v
}

func ShardGetLSeq(key int64) int {
	v,_ := Lshardseqs.Get(key)
	return v
}

func PrintShardSeq()  {
	log.Printf("%v", Shardseqs.GetAll())
	log.Printf("%v", Lshardseqs.GetAll())
}

func ShardIncrement(key int64) int {
	return Shardseqs.Increment(key)
}

//QCP
func ShardIncrementLSeq(key int64) int {
	return Lshardseqs.Increment(key)
}

func ShardUpdateLSeq(key int64, seq int) {
	if seq > ShardGetLSeq(key) {
		Lshardseqs.Insert(key,seq)
	}
}

func ShardUpdateSeq(key int64, seq int) {
	if seq > ShardGetSeq(key) {
		Shardseqs.Insert(key,seq)
	}
}

func ShardResetLSeq(input map[int64]int) {
	Lshardseqs.Set(input)
}

func ShardResetSeq(input map[int64]int) {
	Shardseqs.Set(input)
}

func ShardResetSeqs(){
	Shardseqs.Init()
}

func ShardResetLSeqs(){
	Lshardseqs.Init()
}

func GetShardSeqs() map[int64]int{
	return Shardseqs.GetAll()
}