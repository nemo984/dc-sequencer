package group

import (
	"bytes"
	"encoding/gob"
)

const (
	bufferSize = 1500
)

type MessageType string

const (
	MessageTypeMsg   MessageType = "MESSAGE"
	MessageTypeOrder MessageType = "ORDER"
)

type Message struct {
	Type     MessageType
	ID       string
	Sequence uint32
	Data     map[string]interface{}
}

func (m *Message) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(m); err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func (m *Message) Unmarshal(b []byte, n int) error {
	dec := gob.NewDecoder(bytes.NewReader(b[:n]))
	if err := dec.Decode(&m); err != nil {
		return err
	}
	return nil
}
