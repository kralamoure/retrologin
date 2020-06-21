package d1login

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/kralamoure/d1proto"
	"go.uber.org/zap"
)

type session struct {
	svr  *Server
	conn *net.TCPConn
	salt string
}

func (s *session) receivePkts(ctx context.Context) error {
	rd := bufio.NewReader(s.conn)
	for {
		pkt, err := rd.ReadString('\x00')
		if err != nil {
			return err
		}
		pkt = strings.TrimSuffix(pkt, "\n\x00")
		if pkt == "" {
			continue
		}
		err = s.handlePkt(ctx, pkt)
		if err != nil {
			return err
		}
	}
}

func (s *session) handlePkt(ctx context.Context, pkt string) error {
	id, ok := d1proto.MsgCliIdByPkt(pkt)
	name, _ := d1proto.MsgCliNameByID(id)
	s.svr.logger.Info("received packet from client",
		zap.String("client_address", s.conn.RemoteAddr().String()),
		zap.String("message_name", name),
		zap.String("packet", pkt),
	)
	if !ok {
		return errors.New("unknown packet")
	}
	extra := strings.TrimPrefix(pkt, string(id))
	var err error
	switch id {
	case d1proto.AccountVersion:
		err = s.handleAccountVersion(extra)
	}
	if err != nil {
		return err
	}

	return nil
}

func (s *session) sendMsg(msg d1proto.MsgSvr) error {
	pkt, err := msg.Serialized()
	if err != nil {
		return err
	}
	s.sendPkt(fmt.Sprint(msg.ProtocolId(), pkt))
	return nil
}

func (s *session) sendPkt(pkt string) {
	id, _ := d1proto.MsgSvrIdByPkt(pkt)
	name, _ := d1proto.MsgSvrNameByID(id)
	s.svr.logger.Info("sent packet to client",
		zap.String("client_address", s.conn.RemoteAddr().String()),
		zap.String("message_name", name),
		zap.String("packet", pkt),
	)
	fmt.Fprint(s.conn, pkt+"\x00")
}
