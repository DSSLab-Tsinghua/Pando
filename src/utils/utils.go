package utils

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"sync"
	"time"
	"log"
	"sort"
)

type IntValue struct {
	m int
	sync.RWMutex
}

func (s IntValue) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	ser := IntToBytes(s.m)
	return ser, nil
}

func (s *IntValue) Deserialize(input []byte) error {
	s.Lock()
	s.m = BytesToInt(input)
	defer s.Unlock()
	return nil
}

func (s *IntValue) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = 0
}

func (s *IntValue) Get() int {
	s.Lock()
	defer s.Unlock()
	return s.m
}

func (s *IntValue) Increment() int {
	s.Lock()
	defer s.Unlock()
	s.m = s.m + 1
	return s.m
}

func (s *IntValue) Decrement() int {
	s.Lock()
	defer s.Unlock()
	s.m = s.m - 1
	return s.m
}

func (s *IntValue) Set(n int) {
	s.Lock()
	defer s.Unlock()
	s.m = n
}

func PrintMessage(verbose bool, msg string) {
	if verbose {
		log.Printf(msg)
	}
}

type ByteValue struct {
	m []byte
	sync.RWMutex
}

func (s ByteValue) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return s.m, nil
}

func (s *ByteValue) Deserialize(n []byte) error {
	s.Set(n)
	return nil
}

func (s *ByteValue) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = nil
}

func (s *ByteValue) Get() []byte {
	s.RLock()
	defer s.RUnlock()
	return s.m
}

func (s *ByteValue) Set(n []byte) {
	s.Lock()
	defer s.Unlock()
	s.m = n
}

func BytesToInt(bys []byte) int {
	bytebuff := bytes.NewBuffer(bys)
	var data int64
	binary.Read(bytebuff, binary.BigEndian, &data)
	return int(data)
}

func Int64ToString(input int64) string {
	return strconv.FormatInt(input, 10)
}

func StringToInt(input string) (int, error) {
	return strconv.Atoi(input)
}

func IntToString(input int) string {
	return strconv.Itoa(input)
}

func StringToInt64(input string) (int64, error) {
	return strconv.ParseInt(input, 10, 64)
}

func StringToBytes(input string) []byte {
	return []byte(input)
}

func MakeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func BytesToString(input []byte) string {
	return string(input[:])
}

func IntToInt64(input int) int64 {
	return int64(input)
}

func IntToBytes(n int) []byte {
	data := int64(n)
	bytebuf := bytes.NewBuffer([]byte{})
	binary.Write(bytebuf, binary.BigEndian, data)
	return bytebuf.Bytes()
}

func Int64ToBytes(n int64) []byte {
	data := n
	bytebuf := bytes.NewBuffer([]byte{})
	binary.Write(bytebuf, binary.BigEndian, data)
	return bytebuf.Bytes()
}

func Int64ToInt(input int64) (int, error) {
	tmp := strconv.FormatInt(input, 10)
	output, err := strconv.Atoi(tmp)
	return output, err
}

func SerializeBytes(input [][]byte) []byte {
	if len(input) == 0 {
		return nil
	}
	var output []byte
	for i := 0; i < len(input); i++ {
		output = append(output, input[i]...)
	}
	return output
}

type IntArr struct {
	M []int
	sync.RWMutex
}

func (s *IntArr) GetAll() []int {
	s.RLock()
	defer s.RUnlock()
	return s.M
}

func (s *IntArr) Add(key int) {
	s.Lock()
	defer s.Unlock()
	s.M = append(s.M, key)
}

func (s *IntArr) Set(input []int) {
	s.Lock()
	defer s.Unlock()
	s.M = input
}

func (s *IntArr) Remove(key int) {
	s.Lock()
	defer s.Unlock()
	index := BinarySearch(s.M, key)
	if index < 0 {
		return
	}

	if len(s.M) > index {
		s.M = append(s.M[:index], s.M[index+1:]...)
	} else {
		s.M = s.M[:index]
	}

}

func (s *IntArr) Exists(key int) bool {
	s.RLock()
	defer s.RUnlock()
	index := BinarySearch(s.M, key)
	if index == -1 {
		return false
	}
	return true
}

func (s *IntArr) GetLen() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.M)
}

func (s *IntArr) Sorts() {
	s.Lock()
	defer s.Unlock()
	sort.Ints(s.M)
}

/*
Binary search in an integer array.
Return index of the item, if existed.
Return 0 if the item does not exist.
*/
func BinarySearch(arr []int, key int) int {
	lo, hi := 0, len(arr)-1
	for lo < hi {
		mid := (lo + hi) / 2
		if arr[mid] < key {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	if lo == hi && arr[lo] == key {
		return lo
	}
	return -(lo + 1)
}
