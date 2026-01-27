package bucket

import (
	"pando/src/utils"
)

var (
	KBucketNum			utils.IntValue
	TBucketFailure		utils.IntValue
)

func InitBucket() {
	KBucketNum.Init()
	TBucketFailure.Init()
}