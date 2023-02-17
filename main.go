package main

import (
	"bytes"
	"encoding/gob"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/ipv4"
)

const (
	bufferSize = 1500
)

type GroupMember struct {
	conn      *ipv4.PacketConn
	group     net.Addr
	holdBackQ map[string]*Message // map from msg id to msg
	deliveryQ chan Message
	rg        uint
}

type Message struct {
	Type     string
	ID       string
	Sequence uint
	Data     map[string]interface{}
}

func (gm *GroupMember) sendMulticastMessage(data map[string]interface{}) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(Message{
		ID:   uuid.NewString(),
		Data: data,
	}); err != nil {
		log.Fatal(err)
	}
	b := buf.Bytes()

	log.Printf("multicasting msg %v to the group", data)
	if _, err := gm.conn.WriteTo(b, nil, gm.group); err != nil {
		return err
	}
	return nil
}

func (gm *GroupMember) handleMessage(msg []byte) error {
	return nil
}

func (gm *GroupMember) handleOrderMessage(msg []byte) error {
	return nil
}

func (gm *GroupMember) listen() {
	b := make([]byte, bufferSize)
	for {
		n, cm, src, err := gm.conn.ReadFrom(b)
		if err != nil {
			log.Println(err)
			// error handling
		}
		if cm.Dst.IsMulticast() {
			var msg Message
			dec := gob.NewDecoder(bytes.NewReader(b[:n]))
			if err := dec.Decode(&msg); err != nil {
				log.Fatal(err)
			}
			log.Printf("Received msg: %+v from %v\n", msg, src)

			if msg.Type == "order" {
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

func (gm *GroupMember) handleDeliver() {
	for msg := range gm.deliveryQ {
		log.Printf("Deliverying msg to application, etc: %+v\n", msg)
	}
}

type Sequencer struct {
	conn  *ipv4.PacketConn
	group net.IP
	seq   uint64
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	groupAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5352}
	c, err := net.ListenUDP("udp4", groupAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	eth0, err := net.InterfaceByName("eth0")
	if err != nil {
		log.Fatalf("can't find specified interface %v\n", err)
	}
	pc := ipv4.NewPacketConn(c)
	if err := pc.JoinGroup(eth0, groupAddr); err != nil {
		log.Fatal(err)
	}

	if err := pc.SetControlMessage(ipv4.FlagDst, true); err != nil {
		log.Fatal(err)
	}

	// test
	if loop, err := pc.MulticastLoopback(); err == nil {
		log.Printf("MulticastLoopback status:%v\n", loop)
		if !loop {
			if err := pc.SetMulticastLoopback(true); err != nil {
				log.Printf("SetMulticastLoopback error:%v\n", err)
			}
		}
	}

	gm := &GroupMember{
		conn:      pc,
		group:     &net.UDPAddr{IP: groupAddr.IP, Port: groupAddr.Port},
		holdBackQ: make(map[string]*Message),
		deliveryQ: make(chan Message),
	}

	log.Printf("Group member on %s is connected and listening to group on %s\n", pc.LocalAddr().String(), groupAddr.String())
	ticker := time.NewTicker(2 * time.Second) // TODO: random interval
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				gm.sendMulticastMessage(map[string]interface{}{
					"text": "Hello World",
				})
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	gm.listen()
}
