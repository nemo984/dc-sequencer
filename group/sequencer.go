package group

import (
	"log"
	"net"

	"golang.org/x/net/ipv4"
)

type Sequencer struct {
	conn  *ipv4.PacketConn
	group net.Addr
	seq   uint32
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
			if err := msg.Unmarshal(b, n); err != nil {
				log.Println("unmarshal msg err: ", err)
				continue
			}
			if msg.Type == TypeMsg {
				resMsg := Message{
					Type:     TypeOrder,
					ID:       msg.ID,
					Sequence: s.seq,
				}
				b, _ := resMsg.Marshal()
				log.Printf("multicasting order msg: %+v to the group", resMsg)
				if _, err := s.conn.WriteTo(b, nil, s.group); err != nil {
					log.Println("group mutlicast err: ", err)
				}
				s.seq += 1
			}
		}
	}
}
