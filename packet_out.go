package d1login

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/kralamoure/d1proto"
)

type QueuePacket struct {
	Msg   d1proto.MsgSvr
	Delay time.Duration
}

func (s *Server) HandlePacketsQueue(ctx context.Context, conn *net.TCPConn, queue chan QueuePacket) {
	msgCh := make(chan d1proto.MsgSvr, 0)

	go func() {
		for v := range queue {
			pkt := v
			time.AfterFunc(pkt.Delay, func() {
				msgCh <- pkt.Msg
			})
		}
	}()

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case data := <-msgCh:
			s.SendPacketMsg(conn, data)
		}
	}
}

func (s *Server) SendPacketMsg(conn *net.TCPConn, msg d1proto.MsgSvr) error {
	extra, err := msg.Serialized()
	if err != nil {
		return err
	}

	id := msg.ProtocolId()

	name, ok := d1proto.MsgSvrNameByID(id)
	if !ok {
		name = "Unknown"
	}

	data := string(id) + extra

	s.Logger.Debugw("sent packet",
		"name", name,
		"data", data,
		"address", conn.RemoteAddr().String(),
	)

	_, err = fmt.Fprint(conn, data, "\x00")
	return err
}
