package main

import (
	"context"
	"dc-sequencer/group"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetPrefix("[Sequencer] ")

	groupAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5352}

	c, err := net.ListenUDP("udp4", groupAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close()

	pc, err := group.NewConn(c, groupAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer pc.Close()

	sequencer := group.NewSequencer(pc, groupAddr)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := sequencer.Start(ctx); err != nil {
		log.Println(err)
	}
	log.Println("Terminated")
}
