package group

import (
	"context"
	"errors"
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

// Start spins up message handler for the sequencer.
func (s *Sequencer) Start(ctx context.Context) error {
	log.Println("Sequencer is running inside the group")
	go func() {
		<-ctx.Done()
		s.conn.Close()
	}()

	b := make([]byte, bufferSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, cm, _, err := s.conn.ReadFrom(b)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return ctx.Err()
				}
				log.Println(err)
			}

			if cm.Dst.IsMulticast() {
				msg := &Message{}
				if err := msg.Unmarshal(b, n); err != nil {
					log.Println("unmarshal msg err: ", err)
					continue
				}
				if msg.Type == MessageTypeMsg {
					resMsg := Message{
						Type:     MessageTypeOrder,
						ID:       msg.ID,
						Sequence: s.seq,
					}
					b, _ := resMsg.Marshal()
					log.Printf("multicasting order msg: %+v to the group", resMsg)
					if _, err := s.conn.WriteTo(b, nil, s.group); err != nil {
						log.Println("group mutlicast err: ", err)
					}
					s.seq++
				}
			}
		}
	}
}
