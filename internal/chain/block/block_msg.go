package block

import "encoding/json"

// BlockMsgType defines the type of block message
type BlockMsgType string

// Constants for BlockMsgType
const (
	BlockMsgTypeBlock BlockMsgType = "BLOCK"
	BlockMsgTypeReq   BlockMsgType = "REQ"
	BlockMsgTypeResp  BlockMsgType = "RESP"
)

// BlockMessage represents a message containing a block or a request for a block
type BlockMessage struct {
	Type    BlockMsgType // "BLOCK", "REQ", "RESP"
	Blocks  []*Block     // Type == "BLOCK" or "RESP"
	ReqIdxs uint64       // Type == "REQ"
}

// SerializeBlockMessage serializes a BlockMessage to bytes
func SerializeBlockMessage(msg *BlockMessage) ([]byte, error) {
	return json.Marshal(msg)
}

// DeserializeBlockMessage deserializes bytes into a BlockMessage
func DeserializeBlockMessage(data []byte) (*BlockMessage, error) {
	var msg BlockMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
