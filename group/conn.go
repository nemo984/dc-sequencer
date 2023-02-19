package group

import (
	"log"
	"net"

	"golang.org/x/net/ipv4"
)

func NewConn(c net.PacketConn, group net.Addr) (*ipv4.PacketConn, error) {
	eth0, err := net.InterfaceByName("eth0")
	if err != nil {
		return nil, err
	}

	pc := ipv4.NewPacketConn(c)
	if err := pc.JoinGroup(eth0, group); err != nil {
		return nil, err
	}

	if err := pc.SetControlMessage(ipv4.FlagDst, true); err != nil {
		return nil, err
	}

	if loop, err := pc.MulticastLoopback(); err == nil {
		log.Printf("MulticastLoopback status:%v\n", loop)
		if !loop {
			if err := pc.SetMulticastLoopback(true); err != nil {
				log.Printf("SetMulticastLoopback error:%v\n", err)
			}
		}
	}
	return pc, nil
}
