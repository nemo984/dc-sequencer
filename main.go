package main

import (
	"dc-sequencer/group"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	groupAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5352}
	c, err := net.ListenUDP("udp4", groupAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	pc, err := group.NewConn(c, groupAddr)
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())
	name := fmt.Sprintf("Group-%d", rand.Intn(100))
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	gm := group.NewMember(pc, groupAddr, name)

	log.Printf("Group member on %s is connected and listening to group on %s\n", pc.LocalAddr().String(), groupAddr.String())
	go gm.SendMessages()
	go gm.HandleDeliver()
	gm.Listen()
}
