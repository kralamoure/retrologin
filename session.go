package d1login

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/kralamoure/d1proto"
	"github.com/kralamoure/d1proto/msgcli"
	"github.com/kralamoure/d1proto/msgsvr"
	"go.uber.org/atomic"
	"golang.org/x/time/rate"
)

const (
	statusExpectingAccountVersion uint32 = iota
	statusExpectingAccountCredential
	statusExpectingAccountQueuePosition
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

	accountId string
}

func (s *session) receivePkts(ctx context.Context) error {
	lim := rate.NewLimiter(1, 5)

	rd := bufio.NewReaderSize(s.conn, 256)
	for {
		pkt, err := rd.ReadString('\x00')
		if err != nil {
			return err
		}
		err = lim.Wait(ctx)
		if err != nil {
			return err
		}
		pkt = strings.TrimSuffix(pkt, "\n\x00")
		if pkt == "" {
			continue
		}
		err = s.conn.SetDeadline(time.Now().UTC().Add(s.svr.connTimeout))
		if err != nil {
			return err
		}

		err = s.handlePkt(ctx, pkt)
		if err != nil {
			return err
		}
	}
}

func (s *session) handlePkt(ctx context.Context, pkt string) error {
	defer func() {
		if r := recover(); r != nil {
			s.svr.logger.Errorw("recovered from panic",
				"recover", r,
			)
		}
	}()

	id, ok := d1proto.MsgCliIdByPkt(pkt)
	name, _ := d1proto.MsgCliNameByID(id)
	s.svr.logger.Infow("received packet from client",
		"client_address", s.conn.RemoteAddr().String(),
		"message_name", name,
		"packet", pkt,
	)
	if !ok {
		s.svr.logger.Debugw("unknown packet",
			"client_address", s.conn.RemoteAddr().String(),
		)
		return errEndOfService
	}
	extra := strings.TrimPrefix(pkt, string(id))

	if !s.frameMsg(id) {
		s.svr.logger.Debugw("invalid frame",
			"client_address", s.conn.RemoteAddr().String(),
		)
		return errEndOfService
	}

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
		err := s.handleAccountQueuePosition(ctx)
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
		err := s.handleAccountGetServersList(ctx)
		if err != nil {
			return err
		}
	case d1proto.AccountSetServer:
		msg := msgcli.AccountSetServer{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = s.handleAccountSetServer(ctx, msg)
		if err != nil {
			return err
		}
	default:
		s.sendMsg(msgsvr.BasicsNoticed{})
	}

	return nil
}

func (s *session) frameMsg(id d1proto.MsgCliId) bool {
	status := s.status.Load()
	switch status {
	case statusExpectingAccountVersion:
		if id != d1proto.AccountVersion {
			return false
		}
	case statusExpectingAccountCredential:
		if id != d1proto.AccountCredential {
			return false
		}
	case statusExpectingAccountQueuePosition:
		if id != d1proto.AccountQueuePosition {
			return false
		}
	case statusIdle:
		if id == d1proto.AccountVersion || id == d1proto.AccountCredential {
			return false
		}
	}
	return true
}

func (s *session) sendMsg(msg d1proto.MsgSvr) {
	pkt, err := msg.Serialized()
	if err != nil {
		name, _ := d1proto.MsgSvrNameByID(msg.ProtocolId())
		s.svr.logger.Errorw("could not serialize message",
			"name", name,
		)
		return
	}
	s.sendPkt(fmt.Sprint(msg.ProtocolId(), pkt))
}

func (s *session) sendPkt(pkt string) {
	id, _ := d1proto.MsgSvrIdByPkt(pkt)
	name, _ := d1proto.MsgSvrNameByID(id)
	s.svr.logger.Infow("sent packet to client",
		"client_address", s.conn.RemoteAddr().String(),
		"message_name", name,
		"packet", pkt,
	)
	fmt.Fprint(s.conn, pkt+"\x00")
}
