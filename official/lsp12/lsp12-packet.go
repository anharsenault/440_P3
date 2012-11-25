package lsp12

import (
	"encoding/json"
	"fmt"
)

var typeName = map [byte] string {
	MsgCONNECT: "Connect",
	MsgDATA: "Data",
	MsgACK: "Ack",
	MsgINVALID: "Invalid",
}

// Construct message.  General form
func GenMessage(t byte,  id uint16, seqnum byte, data []byte) *LspMessage {
	return &LspMessage{t, id, seqnum, data}
}

// Construct connection request message.
func GenConnectMessage() *LspMessage {
	return GenMessage(MsgCONNECT, 0, 0, nil)
}

// Construct data message
func GenDataMessage(id uint16, seqnum byte, data []byte) *LspMessage {
	return GenMessage(MsgDATA, id, seqnum, data)
}

// Construct acknowledgment message.
func GenAckMessage(id uint16, seqnum byte) *LspMessage {
	return GenMessage(MsgACK, id, seqnum, nil)
}

// Construct error message
// We will use these to indicate closed connections
func GenInvalidMessage(id uint16, seqnum byte) *LspMessage {
	return GenMessage(MsgINVALID, id, seqnum, nil)	
}

// Extract message from packet
func extractMessage(packet []byte)  (*LspMessage, error) {
	var m LspMessage
	err := json.Unmarshal(packet, &m)
	return &m, err
}

// Pack message into packet
func (msg *LspMessage) genPacket() ([]byte) {
	p, err := json.Marshal(msg)
	if err != nil { return nil }
	return p
}

// Show string representation of message
func (msg *LspMessage) String() string {
	return fmt.Sprintf("[%s %v %v %s]",
		typeName[msg.Type], msg.ConnId, msg.SeqNum,
		string(msg.Payload[0:len(msg.Payload)]))
}

// Determine next sequence number
func NextSeqNum(seqnum byte) byte {
	return seqnum + 1
}
