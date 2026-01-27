/*
Definitions for quorum class
*/

package quorum

import (
	"pando/src/message"
	"pando/src/utils"
	"sync"
)

/*
IntBuffer

	Map int to a set (int64)
	Used to check (given a sequence number) whether a replica receives matching messages from a quorum of replicas

V

	Map int to a list of MessageWithSignature
*/
type INTBUFFER struct {
	IntBuffer map[int]utils.Set
	V         map[int][]message.MessageWithSignature
	sync.RWMutex
}

/*
Buffer

	map string to a set (int64)
	Used to check (given a hash) whether a replica receives matching messages form a quorum of replicas
*/
type BUFFER struct {
	Buffer map[string]utils.Set
	Status bool
	sync.RWMutex
}

// type CERTIFICATE struct {
// 	Certificate map[string][][]byte
// 	Identities  map[string][]int64
// 	sync.RWMutex
// }

/*
Certificate
	Map int (sequence number) to a certificate of a message
*/
type CERTIFICATE struct {
	Certificate map[int]message.Cer
	Identities map[int][]int64
	Vrf map[int]message.Cer
	sync.RWMutex
}

/*
Initialize CERTIFICATE
*/
func (p *CERTIFICATE) Init() {
	p.Lock()
	defer p.Unlock()
	p.Certificate = make(map[int]message.Cer)
	p.Identities = make(map[int][]int64)
	p.Vrf = make(map[int]message.Cer)
}

/*
Add a message to the certificate
Input
	seq: sequence number (int type)
	msg: serialized message ([]byte type)
*/
func (p *CERTIFICATE) Insert(seq int, msg []byte) {
	p.Lock()
	defer p.Unlock()
	var cer message.Cer
	cer, _ = p.Certificate[seq]
	cer.Add(msg)
	p.Certificate[seq] = cer
}

func (p *CERTIFICATE) InsertID(seq int, nid int64) {
	p.Lock()
	defer p.Unlock()
	arr, _ := p.Identities[seq]
	arr = append(arr,nid)
	p.Identities[seq] = arr
}

func (p *CERTIFICATE) InsertVRF(seq int, vrf []byte) {
	p.Lock()
	defer p.Unlock()
	var proof message.Cer
	proof, _ = p.Vrf[seq]
	proof.Add(vrf)
	p.Vrf[seq] = proof
}

/*
Add a certificate to the CERTIFICATE
Input
	seq: sequence number (int type)
	msg: certificate (Cer type)
*/
func (p *CERTIFICATE) Add(seq int, msg message.Cer) {
	p.Lock()
	defer p.Unlock()
	p.Certificate[seq] = msg
}

/*
Get certificate given a sequence number
Input
	seq: sequence number (int type)
Output
	message.Cer: a certificate (Cer type)
	bool: exist or not
*/
func (p *CERTIFICATE) Get(seq int) (message.Cer, bool) {
	p.RLock()
	defer p.RUnlock()
	v, exist := p.Certificate[seq]
	return v, exist
}

func (p *CERTIFICATE) GetID(seq int) ([]int64, bool) {
	p.RLock()
	defer p.RUnlock()
	v, exist := p.Identities[seq]
	return v, exist
}


/*
Size of the certificate given a sequence number
Input
	seq: sequence number (int type)
Output
	int: length of the certificate
*/
func (p *CERTIFICATE) Size(seq int) int {
	p.RLock()
	defer p.RUnlock()
	v, exist := p.Certificate[seq]
	if !exist {
		return 0
	}
	return len(v.Msgs)
}

/*
Delete a certificate given a sequence number
Input
	seq: sequence number (int type)
*/
func (p *CERTIFICATE) Delete(seq int) {
	p.Lock()
	defer p.Unlock()
	delete(p.Certificate, seq)
	delete(p.Identities, seq)
	delete(p.Vrf, seq)
}


// func FetchCer(key string) []byte {
// 	result, exist, result2, exist2 := cer.Get(key)
// 	if !exist || !exist2 {
// 		return nil
// 	}

// 	h := utils.StringToBytes(key)
// 	msg := message.Signatures{
// 		Hash: h,
// 		Sigs: result,
// 		IDs:  result2,
// 	}

// 	msgbyte, err := msg.Serialize()
// 	if err != nil {
// 		return nil
// 	}
// 	return msgbyte
// }

/*
Initialize INTBUFFER
*/
func (b *INTBUFFER) Init(n int) {
	b.IntBuffer = make(map[int]utils.Set)
	b.V = make(map[int][]message.MessageWithSignature)
}

/*
Add a value to INTBUFFER
Input

	input: integer, sequence number (int type)
	id: id of the node (int64 type)
	msg: MessageWithSignature, deserialized message
*/
func (b *INTBUFFER) InsertValue(input int, id int64, msg message.MessageWithSignature) {
	b.Lock()
	defer b.Unlock()
	_, exist := b.IntBuffer[input]
	if exist {
		s := b.IntBuffer[input]
		len1 := s.Len()
		s.AddItem(id)
		b.IntBuffer[input] = s
		len2 := s.Len()
		if len2 > len1 {
			b.V[input] = append(b.V[input], msg)
		}
	} else {
		s := *utils.NewSet()
		s.AddItem(id)
		b.IntBuffer[input] = s
		b.V[input] = append(b.V[input], msg)
	}
}

func (b *INTBUFFER) SetValue(input int, msg []message.MessageWithSignature, ids utils.Set) {
	b.Lock()
	defer b.Unlock()
	b.IntBuffer[input] = ids
	b.V[input] = msg

}

/*
Add value to V only. Used for view changes.
Input

	input: sequence number (int type)
	value: a list of messages
*/
func (b *INTBUFFER) InsertV(input int, value []message.MessageWithSignature) {
	b.Lock()
	defer b.Unlock()
	b.V[input] = value
}

func (b *INTBUFFER) GetV(input int) []message.MessageWithSignature {
	b.Lock()
	defer b.Unlock()
	return b.V[input]
}

/*
Get the number of messages fro IntBuffer
Input

	input: sequence number

Output

	int: number of messages (ids of the nodes)
*/
func (b *INTBUFFER) GetLen(input int) int {
	b.RLock()
	defer b.RUnlock()
	_, exist := b.IntBuffer[input]
	if exist {
		s := b.IntBuffer[input]
		return s.Len()
	}
	return 0
}

/*
Reset INTBUFFER given an input
Input

	input: int type, usually a sequence number
*/
func (b *INTBUFFER) Clear(input int) {
	b.Lock()
	defer b.Unlock()
	_, exist := b.IntBuffer[input]
	if exist {
		delete(b.IntBuffer, input)
	}

	_, exist1 := b.V[input]
	if exist1 {
		delete(b.V, input)
	}
}

/*
Initialize BUFFER
*/
func (b *BUFFER) Init() {
	b.Buffer = make(map[string]utils.Set)
}

func (b *BUFFER) InsertValue(input string, msg []byte, nid int64, step Step, seq int) {
	b.Lock()
	defer b.Unlock()
	_, exist := b.Buffer[input]
	if exist {
		s := b.Buffer[input]
		len := s.Len()
		if len == 0 {
			s = *utils.NewSet()
		}
		s.AddItem(nid)
		b.Buffer[input] = s
		len2 := s.Len()
		if len2 > len {
			if step == PP && seq > 0 {
				PCer.Insert(seq, msg)
				PCer.InsertID(seq, nid)
			}
			if step == QCP && seq > 0 {
				QC.Insert(seq, msg)
				QC.InsertID(seq, nid)
			}
			if step == Pando && seq > 0 {
				QC.Insert(seq, msg)
				QC.InsertID(seq, nid)
			}
		}
	} else {
		s := *utils.NewSet()
		s.AddItem(nid)
		b.Buffer[input] = s
		if step == PP && seq > 0 {
			PCer.Insert(seq, msg)
			PCer.InsertID(seq, nid)
		}
		if step == QCP && seq > 0 {
			QC.Insert(seq, msg)
			QC.InsertID(seq, nid)
		}
		if step == Pando && seq > 0 {
			QC.Insert(seq, msg)
			QC.InsertID(seq, nid)
		}	
	}
}

/*
Get the number of messages in BUFFER
Input

	input: usually a hash (string type)

Output

	int: number of messages
*/
func (b *BUFFER) GetLen(input string) int {
	b.RLock()
	defer b.RUnlock()
	_, exist := b.Buffer[input]
	if exist {
		s := b.Buffer[input]
		return s.Len()
	}
	return 0
}

/*
Reset BUFFER given an input
Input

	input: usually a hash (string type)
*/
func (b *BUFFER) Clear(input string) {
	b.Lock()
	defer b.Unlock()
	_, exist := b.Buffer[input]
	if exist {
		delete(b.Buffer, input)
	}
}

func (b *BUFFER) InsertValueWithVrf(input string, msg []byte, nid int64, step Step, seq int, vrf []byte) {
	b.Lock()
	defer b.Unlock()
	_, exist := b.Buffer[input]
	if exist {
		s := b.Buffer[input]
		len := s.Len()
		if len == 0 {
			s = *utils.NewSet()
		}
		s.AddItem(nid)
		b.Buffer[input] = s
		len2 := s.Len()
		if len2 > len {
			if step == PP && seq > 0 {
				PCer.Insert(seq, msg)
				PCer.InsertID(seq, nid)
				PCer.InsertVRF(seq, vrf)
			}
			if step == Pando && seq > 0 {
				QC.Insert(seq, msg)
				QC.InsertID(seq, nid)
				QC.InsertVRF(seq, vrf)
			}
		}
	} else {
		s := *utils.NewSet()
		s.AddItem(nid)
		b.Buffer[input] = s
		if step == PP && seq > 0 {
			PCer.Insert(seq, msg)
			PCer.InsertID(seq, nid)
			PCer.InsertVRF(seq, vrf)
		}
		if step == Pando && seq > 0 {
			QC.Insert(seq, msg)
			QC.InsertID(seq, nid)
			QC.InsertVRF(seq, vrf)
		}	
	}	
}

func (b *BUFFER) GetValueBykey(key string) (utils.Set, bool) {
	b.Lock()
	defer b.Unlock()
	s, exist := b.Buffer[key]
	return s, exist
}

func (b *BUFFER) InsertValueAndGetlen(input string, msg []byte, nid int64, step Step, seq int) int {
	b.Lock()
	defer b.Unlock()
	_, exist := b.Buffer[input]
	if exist {
		s := b.Buffer[input]
		len := s.Len()
		if len == 0 {
			s = *utils.NewSet()
		}
		s.AddItem(nid)
		b.Buffer[input] = s
		len2 := s.Len()
		if len2 > len {
			if step == PP && seq > 0 {
				PCer.Insert(seq, msg)
			}
			if (step == QCP || step == Pando) && seq > 0 {
				QC.Insert(seq,msg)
			}
		}
	} else {
		s := *utils.NewSet()
		s.AddItem(nid)
		b.Buffer[input] = s
		if step == PP && seq > 0 {
			PCer.Insert(seq, msg)
		}
		if (step == QCP || step == Pando) && seq > 0 {
			QC.Insert(seq,msg)
		}
	}

	_, exi := b.Buffer[input]
	if exi {
		s := b.Buffer[input]
		return s.Len()
	}
	return 0
}

func (b *BUFFER) SetStatus(input string) {
	b.Lock()
	defer b.Unlock()
	b.Status = true
}

func (b *BUFFER) GetStatus(input string) bool {
	b.Lock()
	defer b.Unlock()
	return b.Status
}
