/*
This file implements the buffer of messages and client requests (as queue).
*/

package consensus

import (
	"bytes"
	// "github.com/vmihailenco/msgpack/v5"
	"log"
	"pando/src/config"
	"pando/src/cryptolib"
	pb "pando/src/proto/communication"
	"pando/src/utils"
	"sync"
	"strings"
)

// const maxSize = 400

type QueueHead struct {
	Head  string
	Shard int64 // used for sharding mode. Record the shard number.
	sync.RWMutex
}

func (c *QueueHead) Set(head string) {
	c.Lock()
	defer c.Unlock()
	c.Head = head
}

func (c *QueueHead) SetShard(shardnum int64) {
	c.Lock()
	defer c.Unlock()
	c.Shard = shardnum
}

func (c *QueueHead) ClearShard() {
	c.Lock()
	defer c.Unlock()
	c.Shard = -1
}

func (c *QueueHead) Get() string {
	c.RLock()
	defer c.RUnlock()
	return c.Head
}

func (c *QueueHead) GetShard() int64 {
	c.RLock()
	defer c.RUnlock()
	return c.Shard
}

type Queue struct {
	Q []pb.RawMessage // block
	H map[string]bool //queue helper
	R []int64         // register
	S []int64         // sid
	sync.RWMutex
}

func (q *Queue) Init() {
	q.Q = []pb.RawMessage{}
	//q.H = make(map[string]bool)
	q.R = make([]int64, 0)
	q.S = make([]int64, 0)
}

func (q *Queue) Length() int {
	return len(q.Q)
}

func (q *Queue) Append(item []byte) {
	q.Lock()
	defer q.Unlock()
	q.Q = append(q.Q, pb.RawMessage{Msg: item})
	//h := utils.BytesToString(cryptolib.GenHash(item))
	//q.H[h] = true

}

func (q *Queue) InsertID(value int64) {
	q.Lock()
	defer q.Unlock()
	q.R = append(q.R, value)
}

func (q *Queue) InsertSID(value int64) {
	q.Lock()
	defer q.Unlock()
	q.S = append(q.S, value)
}

func (q *Queue) Grab() []pb.RawMessage {
	q.Lock()
	defer q.Unlock()
	return q.Q
}

func (q *Queue) GrabWithMaxLen() []pb.RawMessage {
	q.Lock()
	defer q.Unlock()
	if len(q.Q) > config.MaxBatchSize() {
		return q.Q[:config.MaxBatchSize()]
	}
	return q.Q
}

//QCP
func (q *Queue) GrabWtihMaxLenAndClear() []pb.RawMessage {
	q.Lock()
	defer q.Unlock()
	if len(q.Q) > config.MaxBatchSize() {
		/*for i:=0; i< config.MaxBatchSize(); i++{
			h := utils.BytesToString(cryptolib.GenHash(q.Q[i].GetMsg()))
			delete(q.H,h)
		}*/
		ret := q.Q[:config.MaxBatchSize()]
		q.Q = q.Q[config.MaxBatchSize():]
		
		return ret
	}
	ret := q.Q
	q.Q = []pb.RawMessage{}
	
	//q.H = make(map[string]bool)
	return ret
}

func (q *Queue) GrabRAndClear() []int64 {
	q.Lock()
	defer q.Unlock()
	ret := q.R
	q.R = make([]int64, 0)
	return ret
}

func (q *Queue) GrabSAndClear() []int64 {
	q.Lock()
	defer q.Unlock()
	ret := q.S
	q.S = make([]int64, 0)
	return ret
}

func (q *Queue) GrabR() []int64 {
	q.Lock()
	defer q.Unlock()
	return q.R
}

func (q *Queue) GrabS() []int64 {
	q.Lock()
	defer q.Unlock()
	return q.S
}

func (q *Queue) GrabRLen() int {
	q.Lock()
	defer q.Unlock()
	return len(q.R)
}

func (q *Queue) GrabSLen() int {
	q.Lock()
	defer q.Unlock()
	return len(q.S)
}

func (q *Queue) GrabQLen() int {
	q.Lock()
	defer q.Unlock()
	return len(q.Q)
}

//QCP
func (q *Queue) GrabFirst() (pb.RawMessage, bool) {
	q.Lock()
	defer q.Unlock()
	if len(q.Q) == 0 {
		var empty pb.RawMessage
		return empty, false
	}
	return q.Q[0], true
}

func (q *Queue) FetchFirst() (string, bool) {
	q.Lock()
	defer q.Unlock()
	if (len(q.Q)) == 0 {
		return "", false
	}
	return utils.BytesToString(cryptolib.GenHash(q.Q[0].GetMsg())), true
}

func (q *Queue) Contains(item pb.RawMessage) (int, bool) {
	q.RLock()
	defer q.RUnlock()
	if len(q.Q) == 0 {
		return 0, false
	}
	for i := 0; i < len(q.Q); i++ {
		if bytes.Compare(item.GetMsg(), q.Q[i].GetMsg()) == 0 {
			return i, true
		}
	}
	/*h := utils.BytesToString(cryptolib.GenHash(item.GetMsg()))
	_,exist := q.H[h]
	if exist{
		return 0, true //todo: the index is not returned
	}*/
	return 0, false
}

//QCP
func (q *Queue) RemoveFirst() {
	q.Lock()
	defer q.Unlock()
	if len(q.Q) == 0 {
		return
	}
	//h := utils.BytesToString(cryptolib.GenHash(q.Q[0].GetMsg()))
	//delete(q.H,h)
	q.Q = q.Q[1:]
}

func (q *Queue) Remove(hash string, msgs []pb.RawMessage) {
	q.Lock()
	defer q.Unlock()

	if len(q.Q) == 0 {
		return
	}

	var tmp []pb.RawMessage
	var tmpmap = make(map[string]bool)
	count := len(msgs)
	marker := 0

	for j := 0; j < len(msgs); j++ {
		marker = j
		h := utils.BytesToString(cryptolib.GenHash(msgs[j].GetMsg()))
		tmpmap[h] = true
		if len(q.Q) <= j {
			continue
		}
		h2 := utils.BytesToString(cryptolib.GenHash(q.Q[j].GetMsg()))
		if strings.Compare(h, h2) == 0 {
			count = count - 1
		} else {
			tmp = append(tmp, q.Q[j])
		}
	}

	for i := len(msgs); i < len(q.Q); i++ {
		marker = i
		if count == 0 {
			//log.Printf("break at index %v, total len %v", marker, len(q.Q))
			break
		}
		h := utils.BytesToString(cryptolib.GenHash(q.Q[i].GetMsg()))
		_, exist := tmpmap[h]
		if exist {
			count = count - 1
			continue
		}
		tmp = append(tmp, q.Q[i])
	}
	if marker >= len(q.Q) {
		q.Q = tmp
		return
	}
	tmp = append(tmp, q.Q[marker:]...)

	/*for i := 0; i < len(q.Q); i++ {
		exist := false
		h := utils.BytesToString(cryptolib.GenHash(q.Q[i].GetMsg()))
		_,exist = q.H[h]
		if !exist{
			continue
		}
		for j := 0; j < len(msgs); j++ {
			item := msgs[j]
			if bytes.Compare(item.GetMsg(), q.Q[i].GetMsg()) == 0 {
				exist = true
			}
		}
		if !exist {
			tmp = append(tmp, q.Q[i])
		}
	}*/
	q.Q = tmp
}

func (q *Queue) RemoveItem(hash []byte) {
	q.Lock()
	defer q.Unlock()

	if len(q.Q) == 0 {
		return
	}

	var tmp []pb.RawMessage
	for i := 0; i < len(q.Q); i++ {
		exist := false
		if bytes.Compare(hash, cryptolib.GenHash(q.Q[i].GetMsg())) == 0 {
			exist = true
		}
		if !exist {
			tmp = append(tmp, q.Q[i])
		}
	}
	q.Q = tmp
}

//QCP
func (q *Queue) IsEmpty() bool {
	return len(q.Q) == 0
}

func (q *Queue) Clear() {
	q.Lock()
	defer q.Unlock()
	q.Q = []pb.RawMessage{}
	q.R = make([]int64, 0)
	q.S = make([]int64, 0)
}

func (q *Queue) ClearFraction(size int) {
	q.Lock()
	defer q.Unlock()
	if len(q.Q) > size {
		q.Q = q.Q[size:]
	} else {
		q.Q = []pb.RawMessage{}
	}
	q.R = make([]int64, 0)
	q.S = make([]int64, 0)
}

func (q *Queue) PrintQueue() {
	for i := 0; i < len(q.Q); i++ {
		log.Println("Number %d: %s", i, q.Q[i].GetMsg())
	}
}
