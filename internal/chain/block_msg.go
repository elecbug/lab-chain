package chain

import "encoding/json"

// BlockMessage represents a message containing a block or a request for a block
type BlockMessage struct {
	Type   string // "BLOCK", "REQ", "RESP"
	Block  *Block // Type == "BLOCK" or "RESP"
	ReqIdx uint64 // Type == "REQ"
}

// serializeBlockMessage serializes a BlockMessage to bytes
func serializeBlockMessage(msg *BlockMessage) ([]byte, error) {
	return json.Marshal(msg)
}

// deserializeBlockMessage deserializes bytes into a BlockMessage
func deserializeBlockMessage(data []byte) (*BlockMessage, error) {
	var msg BlockMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
