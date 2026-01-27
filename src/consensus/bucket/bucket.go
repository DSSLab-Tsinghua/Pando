package bucket



func UpdateKBucketNum(k int) {
	KBucketNum.Set(k)
}

func UpdateTBucketFailure(t int) {
	TBucketFailure.Set(t)
}

func GetKBucketNum() int {
	return KBucketNum.Get()
}

func GetTBucketFailure() int {
	return TBucketFailure.Get()
}
