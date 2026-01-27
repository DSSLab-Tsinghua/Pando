package message

import (
	"github.com/vmihailenco/msgpack/v5"
	pb "pando/src/proto/communication"
)

type ClientRequest struct {
	Type pb.MessageType
	ID   int64
	OP   []byte // Message payload. Opt for contract.
	TS   int64  // Timestamp
}

func (r *ClientRequest) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeClientRequest(input []byte) ClientRequest {
	var clientRequest = new(ClientRequest)
	msgpack.Unmarshal(input, &clientRequest)
	return *clientRequest
}

/*
Reply messages
*/
type ClientReply struct {
	Source int64  `json:"Source"`
	Result bool   `json:"Result"`
	Reply  string `json:"Reply"`
	Msg    []byte `json:"Msg"`
}

/*
Serialize ClientReply
*/
func (r *ClientReply) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

/*
Deserialize []byte to ClientReply
*/
func DeserializeClientReply(input []byte) ClientReply {
	var clientReply = new(ClientReply)
	msgpack.Unmarshal(input, &clientReply)
	return *clientReply
}
