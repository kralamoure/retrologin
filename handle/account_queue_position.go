package handle

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/kralamoure/d1/filter"
	"github.com/kralamoure/d1/typ"
	"github.com/kralamoure/d1proto/enum"
	"github.com/kralamoure/d1proto/msgcli"
	"github.com/kralamoure/d1proto/msgsvr"

	"github.com/kralamoure/d1login"
)

func AccountQueuePosition(s *d1login.Server, sess *d1login.Session, msg msgcli.AccountQueuePosition) error {
	s.SendPacketMsg(sess.Conn, &msgsvr.AccountNewQueue{
		Position:    1,
		TotalAbo:    0,
		TotalNonAbo: 1,
		Subscriber:  false,
		QueueId:     0,
	})

	if sess.Status() == d1login.SessionStatusExpectingFirstQueuePosition {
		var badVersion bool
		if sess.Version.Major != 1 || sess.Version.Minor < 29 {
			badVersion = true
		}

		if badVersion {
			s.SendPacketMsg(sess.Conn, &msgsvr.AccountLoginError{
				Reason: enum.AccountLoginErrorReason.BadVersion,
				Extra:  "^1.29.0",
			})
			return nil
		}

		if sess.Credential.CryptoMethod != 1 {
			return fmt.Errorf("unhandled crypto method: %d", sess.Credential.CryptoMethod)
		}

		password, err := d1login.DecryptedPassword(sess.Credential.Hash, sess.Salt)
		if err != nil {
			return err
		}

		account, err := s.Login.Account(filter.AccountNameEQ(typ.AccountName(sess.Credential.Username)))
		if err != nil {
			s.SendPacketMsg(sess.Conn, &msgsvr.AccountLoginError{
				Reason: enum.AccountLoginErrorReason.AccessDenied,
			})
			return err
		}

		user, err := s.Login.User(filter.UserIdEQ(account.UserId))
		if err != nil {
			return err
		}

		match, err := argon2id.ComparePasswordAndHash(password, string(user.Hash))
		if err != nil {
			return err
		}

		if !match {
			s.SendPacketMsg(sess.Conn, &msgsvr.AccountLoginError{
				Reason: enum.AccountLoginErrorReason.AccessDenied,
			})
			return errors.New("wrong password")
		}

		sess.LastAccess = account.LastAccess
		sess.LastIP = account.LastIP

		ip, _, err := net.SplitHostPort(sess.Conn.RemoteAddr().String())
		if err != nil {
			return err
		}
		s.Login.SetAccountLastAccessAndIP(account.Id, time.Now(), ip)

		s.DeleteSessionByAccountId(account.Id)

		sess.AccountId = account.Id

		s.SendPacketMsg(sess.Conn, &msgsvr.AccountPseudo{Value: string(user.Nickname)})
		s.SendPacketMsg(sess.Conn, &msgsvr.AccountCommunity{Id: int(user.Community)})

		hosts := &msgsvr.AccountHosts{}
		err = hosts.Deserialize(s.HostsData())
		if err != nil {
			return err
		}
		s.SendPacketMsg(sess.Conn, hosts)

		s.SendPacketMsg(sess.Conn, &msgsvr.AccountLoginSuccess{Authorized: account.Admin})
		s.SendPacketMsg(sess.Conn, &msgsvr.AccountSecretQuestion{Value: "5 + 6"})

		sess.SetStatus(d1login.SessionStatusIdle)
	}

	return nil
}
