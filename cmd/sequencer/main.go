package main

import (
	"dc-sequencer/group"
	"log"
	"net"
)

func main() {
	groupAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5352}

	c2, err := net.ListenUDP("udp4", groupAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer c2.Close()
	pc2, err := group.NewConn(c2, groupAddr)
	if err != nil {
		log.Fatal(err)
	}

	sequencer := group.NewSequencer(pc2, groupAddr)
	go sequencer.Listen()
	log.Println("Sequencer is running inside the group")
	select {}
}
