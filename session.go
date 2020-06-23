package d1login

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/kralamoure/d1proto"
	"github.com/kralamoure/d1proto/msgcli"
	"github.com/kralamoure/d1proto/msgsvr"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

const (
	statusExpectingVersion uint32 = iota
	statusExpectingCredential
	statusExpectingQueuePosition
	statusIdle
)

var errEndOfService = errors.New("end of service")

type session struct {
	svr    *Server
	conn   *net.TCPConn
	salt   string
	status atomic.Uint32

	version    msgcli.AccountVersion
	credential msgcli.AccountCredential

	accountId int
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
	s.svr.logger.Debug("received packet from client",
		zap.String("client_address", s.conn.RemoteAddr().String()),
		zap.String("message_name", name),
		zap.String("packet", pkt),
	)
	if !ok {
		return errors.New("unknown packet")
	}
	extra := strings.TrimPrefix(pkt, string(id))

	switch id {
	case d1proto.AccountVersion:
		msg := msgcli.AccountVersion{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = s.handleAccountVersion(msg)
		if err != nil {
			return err
		}
	case d1proto.AccountCredential:
		msg := msgcli.AccountCredential{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = s.handleAccountCredential(msg)
		if err != nil {
			return err
		}
	case d1proto.AccountQueuePosition:
		msg := msgcli.AccountQueuePosition{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = s.handleAccountQueuePosition(ctx, msg)
		if err != nil {
			return err
		}
	case d1proto.AccountSearchForFriend:
		msg := msgcli.AccountSearchForFriend{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = s.handleAccountSearchForFriend(ctx, msg)
		if err != nil {
			return err
		}
	case d1proto.AccountGetServersList:
		msg := msgcli.AccountGetServersList{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = s.AccountGetServersList(ctx, msg)
		if err != nil {
			return err
		}
	case d1proto.AccountSetServer:
		msg := msgcli.AccountSetServer{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = s.AccountSetServer(ctx, msg)
		if err != nil {
			return err
		}
	default:
		s.sendMsg(msgsvr.BasicsNoticed{})
	}

	return nil
}

func (s *session) sendMsg(msg d1proto.MsgSvr) {
	pkt, err := msg.Serialized()
	if err != nil {
		name, _ := d1proto.MsgSvrNameByID(msg.ProtocolId())
		s.svr.logger.Error("could not serialize message",
			zap.String("name", name),
		)
		return
	}
	s.sendPkt(fmt.Sprint(msg.ProtocolId(), pkt))
}

func (s *session) sendPkt(pkt string) {
	id, _ := d1proto.MsgSvrIdByPkt(pkt)
	name, _ := d1proto.MsgSvrNameByID(id)
	s.svr.logger.Debug("sent packet to client",
		zap.String("client_address", s.conn.RemoteAddr().String()),
		zap.String("message_name", name),
		zap.String("packet", pkt),
	)
	fmt.Fprint(s.conn, pkt+"\x00")
}
