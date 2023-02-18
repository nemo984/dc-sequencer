package group

import (
	"log"
	"net"

	"golang.org/x/net/ipv4"
)

type Sequencer struct {
	conn  *ipv4.PacketConn
	group net.Addr
	seq   uint64
}

func NewSequencer(conn *ipv4.PacketConn, group net.Addr) *Sequencer {
	return &Sequencer{
		conn:  conn,
		group: group,
	}
}

func (s *Sequencer) Listen() {
	b := make([]byte, bufferSize)
	for {
		n, cm, _, err := s.conn.ReadFrom(b)
		if err != nil {
			log.Println(err)
		}
		if cm.Dst.IsMulticast() {
			msg := &Message{}
			_ = msg.Unmarshal(b, n)
			log.Printf("Received msg: %+v from the group as sequencer\n", msg)

			if msg.Type == TypeMsg {
				// sequencer order
			}
		}
	}
}
