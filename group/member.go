package group

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/ipv4"
)

type GroupMember struct {
	conn      *ipv4.PacketConn
	group     net.Addr
	name      string   // for identifying member
	holdBackQ sync.Map // map from msg id to msg

	rg    uint32
	rgMtx sync.Mutex

	deliveryQ chan *Message
	out       io.Writer // output for msg delivery
}

func NewMember(name string, conn *ipv4.PacketConn, group net.Addr, out io.Writer) *GroupMember {
	return &GroupMember{
		conn:      conn,
		group:     group,
		holdBackQ: sync.Map{},
		deliveryQ: make(chan *Message),
		name:      name,
		out:       out,
	}
}

func (gm *GroupMember) SendMulticastMessage(data map[string]interface{}) error {
	msg := &Message{
		ID:   uuid.NewString(),
		Data: data,
		Type: TypeMsg,
	}
	b, _ := msg.Marshal()
	log.Printf("multicasting msg %v to the group", data)
	if _, err := gm.conn.WriteTo(b, nil, gm.group); err != nil {
		return err
	}
	return nil
}

func (gm *GroupMember) Listen() {
	b := make([]byte, bufferSize)
	for {
		n, cm, _, err := gm.conn.ReadFrom(b)
		if err != nil {
			log.Println(err)
		}

		if cm.Dst.IsMulticast() {
			recvMsg := Message{}
			if err := recvMsg.Unmarshal(b, n); err != nil {
				log.Println("unmarshal error", err)
				continue
			}
			log.Printf("Received msg: %+v from the group\n", recvMsg)

			if recvMsg.Type == TypeOrder {
				// sequencer order
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				go func(ctx context.Context) {
					for {
						select {
						case <-ctx.Done():
							return
						default:
							if m, ok := gm.holdBackQ.Load(recvMsg.ID); ok && recvMsg.Sequence == gm.rg {
								msg := m.(*Message)
								// msg is in hold back queue
								msg.Sequence = recvMsg.Sequence
								gm.holdBackQ.Delete(msg.ID)
								// TO-deliver m;
								gm.deliveryQ <- msg
								gm.rgMtx.Lock()
								gm.rg = recvMsg.Sequence + 1
								gm.rgMtx.Unlock()
								return
							} else {
								// wait until <m, i> in hold-back queue and S = rg;
								time.Sleep(100 * time.Millisecond)
							}
						}
					}
				}(ctx)
			} else {
				// Place msg in hold-back queue
				gm.holdBackQ.Store(recvMsg.ID, &recvMsg)
			}
		}
	}
}

func (gm *GroupMember) SendMessages() {
	// periodically and randomly send messgaes
	ticker := time.NewTicker(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			gm.SendMulticastMessage(map[string]interface{}{
				"text":      fmt.Sprintf("Hello from %s", gm.name),
				"timestamp": time.Now().Format("15:04:05"),
			})
			ticker.Reset(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func (gm *GroupMember) HandleDeliver() {
	for msg := range gm.deliveryQ {
		log.Printf("Deliverying Msg ID: %s\n", msg.ID)
		if gm.out != nil {
			fmt.Fprintf(gm.out, "Sequence: %d, ID:%s, Data:%s\n", msg.Sequence, msg.ID, msg.Data)
		}
	}
}
