package main

import (
	"context"
	"dc-sequencer/group"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultMemberName = "member-%d"
	outputFileName    = "output-%s.txt"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	groupAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5352}
	c, err := net.ListenUDP("udp4", groupAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close()

	pc, err := group.NewConn(c, groupAddr)
	if err != nil {
		log.Println("NewConn error:", err)
		return
	}
	defer pc.Close()

	rand.Seed(time.Now().UnixNano())

	name := fmt.Sprintf(defaultMemberName, rand.Intn(100))
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	log.SetPrefix(fmt.Sprintf("[%s] ", name))

	f, err := os.OpenFile(fmt.Sprintf(outputFileName, name), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	gm := group.NewMember(name, pc, groupAddr, f)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := gm.Start(ctx); err != nil {
		log.Println(err)
	}
	log.Println("Terminated gracefully")
}
