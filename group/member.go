package group

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/ipv4"
)

type GroupMember struct {
	conn      *ipv4.PacketConn
	group     net.Addr
	holdBackQ map[string]*Message // map from msg id to msg
	deliveryQ chan Message
	rg        uint
	name      string // just for identifying member - printing
}

func NewMember(conn *ipv4.PacketConn, group net.Addr, name string) *GroupMember {
	return &GroupMember{
		conn:      conn,
		group:     group,
		holdBackQ: make(map[string]*Message),
		deliveryQ: make(chan Message),
		name:      name,
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
			// error handling
		}
		if cm.Dst.IsMulticast() {
			msg := Message{}
			_ = msg.Unmarshal(b, n)
			log.Printf("Received msg: %+v from the group\n", msg)

			if msg.Type == TypeOrder {
				// sequencer order
				// wait until <m, i> in hold-back queue and S = rg;
				go func() {
					// TO-deliver m;
					gm.deliveryQ <- msg
					gm.rg = msg.Sequence + 1
				}()
			} else {
				// Place msg in hold-back queue
				gm.holdBackQ[msg.ID] = &msg
			}
		}
	}
}

func (gm *GroupMember) SendMessages() {
	// periodically and renadom send messgaes
	ticker := time.NewTicker(2 * time.Second) // TODO: random interval
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			gm.SendMulticastMessage(map[string]interface{}{
				"text": fmt.Sprintf("Hello from %s", gm.name),
			})
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func (gm *GroupMember) HandleDeliver() {
	for msg := range gm.deliveryQ {
		log.Printf("Deliverying msg to application, etc: %+v\n", msg)
	}
}
