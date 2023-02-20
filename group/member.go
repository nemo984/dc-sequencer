package group

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/ipv4"
	"golang.org/x/sync/errgroup"
)

type Member struct {
	conn      *ipv4.PacketConn
	group     net.Addr
	name      string   // for identifying member
	holdBackQ sync.Map // map from msg id to msg
	rg        uint32

	deliveryQ chan *Message
	out       io.Writer // output for msg delivery
}

func NewMember(name string, conn *ipv4.PacketConn, group net.Addr, out io.Writer) *Member {
	return &Member{
		conn:      conn,
		group:     group,
		holdBackQ: sync.Map{},
		deliveryQ: make(chan *Message),
		name:      name,
		out:       out,
	}
}

// Start spins up all the goroutines to handle messages and for sending them.
func (gm *Member) Start(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return gm.handleMessages(gctx)
	})
	g.Go(func() error {
		return gm.sendMessages(gctx)
	})
	g.Go(func() error {
		gm.handleDeliver()
		return nil
	})
	g.Go(func() error {
		<-gctx.Done()
		close(gm.deliveryQ)
		return gm.conn.Close()
	})
	log.Printf("Group member on %s is connected and listening to group on %s\n", gm.conn.LocalAddr().String(), gm.group.String())

	return g.Wait()
}

// multicast multicast message to the group.
func (gm *Member) multicast(data map[string]interface{}) error {
	msg := &Message{
		ID:   uuid.NewString(),
		Data: data,
		Type: MessageTypeMsg,
	}
	b, _ := msg.Marshal()
	log.Printf("multicasting msg %v to the group", data)
	if _, err := gm.conn.WriteTo(b, nil, gm.group); err != nil {
		return err
	}
	return nil
}

// handleMessages handles the message received from the group.
func (gm *Member) handleMessages(ctx context.Context) error {
	b := make([]byte, bufferSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, cm, _, err := gm.conn.ReadFrom(b)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return nil
				}
				log.Println(err)
			}

			if cm.Dst.IsMulticast() {
				recvMsg := Message{}
				if err := recvMsg.Unmarshal(b, n); err != nil {
					log.Println("unmarshal error", err)
					continue
				}
				log.Printf("Received msg: %+v from the group\n", recvMsg)

				if recvMsg.Type == MessageTypeOrder {
					// sequencer order
					ctx, cancel := context.WithTimeout(ctx, time.Second*3)
					defer cancel()

					go func(ctx context.Context) {
						for {
							select {
							case <-ctx.Done():
								return
							default:
								if m, ok := gm.holdBackQ.Load(recvMsg.ID); ok && recvMsg.Sequence == atomic.LoadUint32(&gm.rg) {
									msg := m.(*Message)
									// msg is in hold back queue
									msg.Sequence = recvMsg.Sequence
									gm.holdBackQ.Delete(msg.ID)
									// TO-deliver m;
									gm.deliveryQ <- msg
									atomic.StoreUint32(&gm.rg, recvMsg.Sequence+1)
									return
								}
								// until <m, i> in hold-back queue and S = rg;
								time.Sleep(100 * time.Millisecond)
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
}

// sendMessages periodically and randomly send messages to the group.
func (gm *Member) sendMessages(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			if err := gm.multicast(map[string]interface{}{
				"text":      fmt.Sprintf("Hello from %s", gm.name),
				"timestamp": time.Now().Format("15:04:05"),
			}); err != nil {
				return err
			}
			ticker.Reset(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)
		case <-ctx.Done():
			ticker.Stop()
			return ctx.Err()
		}
	}
}

// handleDeliver delivers message inside the delivery queue.
func (gm *Member) handleDeliver() {
	for msg := range gm.deliveryQ {
		log.Printf("Delivering Msg ID: %s\n", msg.ID)
		if gm.out != nil {
			fmt.Fprintf(gm.out, "Sequence: %d, ID:%s, Data:%s\n", msg.Sequence, msg.ID, msg.Data)
		}
	}
}
