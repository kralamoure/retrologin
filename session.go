package d1login

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/kralamoure/d1proto"
	"github.com/kralamoure/d1proto/enum"
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

type msgOut interface {
	ProtocolId() (id d1proto.MsgSvrId)
	Serialized() (extra string, err error)
}

func (s *session) receivePackets(ctx context.Context) error {
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

		err = s.handlePacket(ctx, pkt)
		if err != nil {
			return err
		}
	}
}

func (s *session) handlePacket(ctx context.Context, pkt string) error {
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

	if !s.frameMessage(id) {
		s.svr.logger.Debugw("invalid frame",
			"client_address", s.conn.RemoteAddr().String(),
		)
		return errEndOfService
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
		s.sendMessage(msgsvr.BasicsNothing{})
	}

	return nil
}

func (s *session) frameMessage(id d1proto.MsgCliId) bool {
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

func (s *session) login(ctx context.Context) error {
	if s.version.Major != 1 || s.version.Minor < 29 {
		s.sendMessage(msgsvr.AccountLoginError{
			Reason: enum.AccountLoginErrorReason.BadVersion,
			Extra:  "^1.29.0",
		})
		versionStr, err := s.version.Serialized()
		if err != nil {
			return err
		}
		s.svr.logger.Debugw("wrong version",
			"client_address", s.conn.RemoteAddr().String(),
			"version", versionStr,
		)
		return errEndOfService
	}

	if s.credential.CryptoMethod != 1 {
		s.svr.logger.Debugw("unhandled crypto method",
			"client_address", s.conn.RemoteAddr().String(),
			"crypto_method", s.credential.CryptoMethod,
		)
		return errEndOfService
	}

	password, err := decryptedPassword(s.credential.Hash, s.salt)
	if err != nil {
		s.svr.logger.Debugw("could not decrypt password",
			"error", err,
			"client_address", s.conn.RemoteAddr().String(),
		)
		return errEndOfService
	}

	account, err := s.svr.dofus.AccountByName(ctx, s.credential.Username)
	if err != nil {
		s.sendMessage(msgsvr.AccountLoginError{
			Reason: enum.AccountLoginErrorReason.AccessDenied,
		})
		if errors.Is(err, d1proto.ErrNotFound) {
			s.svr.logger.Debugw("could not find account",
				"error", err,
				"client_address", s.conn.RemoteAddr().String(),
			)
			return nil
		} else {
			return err
		}
	}

	user, err := s.svr.dofus.User(ctx, account.UserId)
	if err != nil {
		return err
	}

	match, err := argon2id.ComparePasswordAndHash(password, string(user.Hash))
	if err != nil {
		return err
	}

	if !match {
		s.sendMessage(msgsvr.AccountLoginError{
			Reason: enum.AccountLoginErrorReason.AccessDenied,
		})
		s.svr.logger.Debugw("wrong password",
			"client_address", s.conn.RemoteAddr().String(),
		)
		return errEndOfService
	}

	err = s.svr.controlAccount(account.Id, s)
	if err != nil {
		s.sendMessage(msgsvr.AccountLoginError{
			Reason: enum.AccountLoginErrorReason.AlreadyLogged,
		})
		s.svr.logger.Debugw("could not control account",
			"error", err,
			"client_address", s.conn.RemoteAddr().String(),
		)
		return errEndOfService
	}
	s.accountId = account.Id

	s.sendMessage(msgsvr.AccountPseudo{Value: string(user.Nickname)})
	s.sendMessage(msgsvr.AccountCommunity{Id: int(user.Community)})
	s.sendMessage(msgsvr.AccountSecretQuestion{Value: user.SecretQuestion})

	hosts := msgsvr.AccountHosts{}
	err = hosts.Deserialize(s.svr.hosts.Load())
	if err != nil {
		return err
	}
	s.sendMessage(hosts)

	s.sendMessage(msgsvr.AccountLoginSuccess{Authorized: account.Admin})

	s.status.Store(statusIdle)
	return nil
}

func (s *session) sendMessage(msg msgOut) {
	pkt, err := msg.Serialized()
	if err != nil {
		name, _ := d1proto.MsgSvrNameByID(msg.ProtocolId())
		s.svr.logger.Errorw("could not serialize message",
			"name", name,
		)
		return
	}
	s.sendPacket(fmt.Sprint(msg.ProtocolId(), pkt))
}

func (s *session) sendPacket(pkt string) {
	id, _ := d1proto.MsgSvrIdByPkt(pkt)
	name, _ := d1proto.MsgSvrNameByID(id)
	s.svr.logger.Infow("sent packet to client",
		"client_address", s.conn.RemoteAddr().String(),
		"message_name", name,
		"packet", pkt,
	)
	fmt.Fprint(s.conn, pkt+"\x00")
}
