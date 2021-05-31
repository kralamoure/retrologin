package retrologin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/kralamoure/retroproto"
	"github.com/kralamoure/retroproto/enum"
	"github.com/kralamoure/retroproto/msgcli"
	"github.com/kralamoure/retroproto/msgsvr"
	"go.uber.org/atomic"
	"golang.org/x/time/rate"
)

const (
	statusExpectingAccountVersion uint32 = iota
	statusExpectingAccountCredential
	statusExpectingAccountQueuePosition
	statusIdle
)

var errInvalidRequest = errors.New("invalid request")

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
	ProtocolId() (id retroproto.MsgSvrId)
	Serialized() (extra string, err error)
}

func (s *session) receivePackets(ctx context.Context) error {
	lim := rate.NewLimiter(1, 5)

	rd := bufio.NewReaderSize(s.conn, 256)
	for {
		pkt, err := rd.ReadString('\x00')
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				s.sendMessage(msgsvr.AksServerMessage{Value: "01"})
			}
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
		err = s.conn.SetReadDeadline(time.Now().UTC().Add(s.svr.connTimeout))
		if err != nil {
			return err
		}

		err = s.handlePacket(pkt)
		if err != nil {
			return err
		}
	}
}

func (s *session) handlePacket(pkt string) error {
	defer func() {
		if r := recover(); r != nil {
			s.svr.logger.Errorw("recovered from panic",
				"recover", r,
			)
		}
	}()

	id, ok := retroproto.MsgCliIdByPkt(pkt)
	name, _ := retroproto.MsgCliNameByID(id)
	s.svr.logger.Infow("received packet from client",
		"client_address", s.conn.RemoteAddr().String(),
		"message_name", name,
		"packet", pkt,
	)
	if !ok {
		s.svr.logger.Debugw("unknown packet",
			"client_address", s.conn.RemoteAddr().String(),
		)
		return errInvalidRequest
	}
	extra := strings.TrimPrefix(pkt, string(id))

	if !s.frameMessage(id) {
		s.svr.logger.Debugw("invalid frame",
			"client_address", s.conn.RemoteAddr().String(),
		)
		return errInvalidRequest
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch id {
	case retroproto.AccountVersion:
		msg := msgcli.AccountVersion{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = s.handleAccountVersion(msg)
		if err != nil {
			return err
		}
	case retroproto.AccountCredential:
		msg := msgcli.AccountCredential{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = s.handleAccountCredential(msg)
		if err != nil {
			return err
		}
	case retroproto.AccountQueuePosition:
		err := s.handleAccountQueuePosition(ctx)
		if err != nil {
			return err
		}
	case retroproto.AccountSearchForFriend:
		msg := msgcli.AccountSearchForFriend{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = s.handleAccountSearchForFriend(ctx, msg)
		if err != nil {
			return err
		}
	case retroproto.AccountGetServersList:
		err := s.handleAccountGetServersList(ctx)
		if err != nil {
			return err
		}
	case retroproto.AccountSetServer:
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

func (s *session) frameMessage(id retroproto.MsgCliId) bool {
	status := s.status.Load()
	switch status {
	case statusExpectingAccountVersion:
		if id != retroproto.AccountVersion {
			return false
		}
	case statusExpectingAccountCredential:
		if id != retroproto.AccountCredential {
			return false
		}
	case statusExpectingAccountQueuePosition:
		if id != retroproto.AccountQueuePosition {
			return false
		}
	case statusIdle:
		if id == retroproto.AccountVersion || id == retroproto.AccountCredential {
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
		return errInvalidRequest
	}

	if s.credential.CryptoMethod != 1 {
		s.svr.logger.Debugw("unhandled crypto method",
			"client_address", s.conn.RemoteAddr().String(),
			"crypto_method", s.credential.CryptoMethod,
		)
		return errInvalidRequest
	}

	password, err := decryptedPassword(s.credential.Hash, s.salt)
	if err != nil {
		s.svr.logger.Debugw("could not decrypt password",
			"error", err,
			"client_address", s.conn.RemoteAddr().String(),
		)
		return errInvalidRequest
	}

	account, err := s.svr.dofus.AccountByName(ctx, s.credential.Username)
	if err != nil {
		s.sendMessage(msgsvr.AccountLoginError{
			Reason: enum.AccountLoginErrorReason.AccessDenied,
		})
		if errors.Is(err, retroproto.ErrNotFound) {
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
		return errInvalidRequest
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
		return errInvalidRequest
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
		name, _ := retroproto.MsgSvrNameByID(msg.ProtocolId())
		s.svr.logger.Errorw("could not serialize message",
			"name", name,
		)
		return
	}
	s.sendPacket(fmt.Sprint(msg.ProtocolId(), pkt))
}

func (s *session) sendPacket(pkt string) {
	id, _ := retroproto.MsgSvrIdByPkt(pkt)
	name, _ := retroproto.MsgSvrNameByID(id)
	s.svr.logger.Infow("sent packet to client",
		"client_address", s.conn.RemoteAddr().String(),
		"message_name", name,
		"packet", pkt,
	)
	fmt.Fprint(s.conn, pkt+"\x00")
}
