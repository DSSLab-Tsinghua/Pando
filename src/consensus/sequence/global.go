package sequence

import (
	"pando/src/utils"
)

var (
	Sequence   utils.IntValue //current Sequence number
	Lsequence  utils.IntValue
	Shardseqs  utils.Int64IntMap // used for sharding mode, tracking the committed Sequence numbers for each shard
	Lshardseqs utils.Int64IntMap //used for sharding mode at the leader, tracking the proposed Sequence numbers for each shard


	//only used in archive node
	leastSeq utils.IntValue

	leastShardSeq utils.Int64IntMap
	Epoch   utils.IntValue
	LEpoch   utils.IntValue
)

func InitSequence() {

	Sequence.Init()
	Lsequence.Init()
	Shardseqs.Init()
	Lshardseqs.Init()

	leastSeq.Init()
	leastShardSeq.Init()

	Epoch.Init()
	LEpoch.Init()
}
