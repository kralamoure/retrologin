package d1login

import (
	"fmt"

	"github.com/kralamoure/d1proto/enum"
	"github.com/kralamoure/d1proto/msgcli"
	"github.com/kralamoure/d1proto/msgsvr"
)

func (s *session) handleAccountVersion(m msgcli.AccountVersion) error {
	/*sess.Version = m
	sess.SetStatus(d1login.SessionStatusExpectingCredential)*/

	s.version = m
	s.status.Store(statusExpectingCredential)
	return nil
}

func (s *session) handleAccountCredential(m msgcli.AccountCredential) error {
	/*sess.Credential = m
	sess.SetStatus(d1login.SessionStatusExpectingFirstQueuePosition)*/

	s.credential = m
	s.status.Store(statusExpectingQueuePosition)
	return nil
}

func (s *session) handleAccountQueuePosition(m msgcli.AccountQueuePosition) error {
	/*s.SendPacketMsg(sess.conn, &msgsvr.AccountNewQueue{
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
			s.SendPacketMsg(sess.conn, &msgsvr.AccountLoginError{
				Reason: enum.AccountLoginErrorReason.BadVersion,
				Extra:  "^1.29.0",
			})
			return nil
		}

		if sess.Credential.CryptoMethod != 1 {
			return fmt.Errorf("unhandled crypto method: %d", sess.Credential.CryptoMethod)
		}

		password, err := d1login.decryptedPassword(sess.Credential.Hash, sess.salt)
		if err != nil {
			return err
		}

		account, err := s.Login.Account(filter.AccountNameEQ(typ.AccountName(sess.Credential.Username)))
		if err != nil {
			s.SendPacketMsg(sess.conn, &msgsvr.AccountLoginError{
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
			s.SendPacketMsg(sess.conn, &msgsvr.AccountLoginError{
				Reason: enum.AccountLoginErrorReason.AccessDenied,
			})
			return errors.New("wrong password")
		}

		sess.LastAccess = account.LastAccess
		sess.LastIP = account.LastIP

		ip, _, err := net.SplitHostPort(sess.conn.RemoteAddr().String())
		if err != nil {
			return err
		}
		s.Login.SetAccountLastAccessAndIP(account.Id, time.Now(), ip)

		s.DeleteSessionByAccountId(account.Id)

		sess.AccountId = account.Id

		s.SendPacketMsg(sess.conn, &msgsvr.AccountPseudo{Value: string(user.Nickname)})
		s.SendPacketMsg(sess.conn, &msgsvr.AccountCommunity{Id: int(user.Community)})

		hosts := &msgsvr.AccountHosts{}
		err = hosts.Deserialize(s.HostsData())
		if err != nil {
			return err
		}
		s.SendPacketMsg(sess.conn, hosts)

		s.SendPacketMsg(sess.conn, &msgsvr.AccountLoginSuccess{Authorized: account.Admin})
		s.SendPacketMsg(sess.conn, &msgsvr.AccountSecretQuestion{Value: "5 + 6"})

		sess.SetStatus(d1login.SessionStatusIdle)
	}*/

	err := s.sendMsg(msgsvr.AccountNewQueue{
		Position:    1,
		TotalAbo:    0,
		TotalNonAbo: 1,
		Subscriber:  false,
		QueueId:     0,
	})
	if err != nil {
		return err
	}

	if s.status.Load() == statusExpectingQueuePosition {
		if s.version.Major != 1 || s.version.Minor < 29 {
			err := s.sendMsg(msgsvr.AccountLoginError{
				Reason: enum.AccountLoginErrorReason.BadVersion,
				Extra:  "^1.29.0",
			})
			if err != nil {
				return err
			}
			return nil
		}

		if s.credential.CryptoMethod != 1 {
			return fmt.Errorf("unhandled crypto method: %d", s.credential.CryptoMethod)
		}

		password, err := decryptedPassword(s.credential.Hash, s.salt)
		if err != nil {
			return err
		}
		s.svr.logger.Debug(password)
		s.status.Store(statusIdle)
	}

	return nil
}

func (s *session) handleAccountSearchForFriend(extra string) error {
	/*user, err := s.Login.User(filter.UserNicknameEQ(typ.Nickname(msg.Pseudo)))
	if err != nil {
		if errors.Is(err, d1.ErrResourceNotFound) {
			s.SendPacketMsg(sess.conn, &msgsvr.AccountFriendServerList{})
			return nil
		} else {
			return err
		}
	}

	accounts, err := s.Login.Accounts(filter.AccountUserIdEQ(user.Id))
	if err != nil {
		return err
	}

	serverIdQty := make(map[int]int)

	for _, account := range accounts {
		characters, err := s.Login.Characters(filter.CharacterAccountIdEQ(account.Id))
		if err != nil {
			return err
		}
		for _, character := range characters {
			serverIdQty[character.GameServerId]++
		}
	}

	var serverCharacters []typ2.AccountServersListServerCharacters

	for serverId, qty := range serverIdQty {
		serverCharacters = append(serverCharacters, typ2.AccountServersListServerCharacters{
			Id:  serverId,
			Qty: qty,
		})
	}

	sort.Slice(serverCharacters, func(i, j int) bool { return serverCharacters[i].Id < serverCharacters[j].Id })

	s.SendPacketMsg(sess.conn, &msgsvr.AccountFriendServerList{ServersCharacters: serverCharacters})*/

	return nil
}

func (s *session) AccountGetServersList(extra string) error {
	/*account, err := s.Login.Account(filter.AccountIdEQ(sess.AccountId))
	if err != nil {
		return err
	}

	serverIdQty := make(map[int]int)

	characters, err := s.Login.Characters(filter.CharacterAccountIdEQ(sess.AccountId))
	if err != nil {
		return err
	}
	for _, character := range characters {
		serverIdQty[character.GameServerId]++
	}

	var serverCharacters []typ2.AccountServersListServerCharacters

	for serverId, qty := range serverIdQty {
		serverCharacters = append(serverCharacters, typ2.AccountServersListServerCharacters{
			Id:  serverId,
			Qty: qty,
		})
	}

	sort.Slice(serverCharacters, func(i, j int) bool { return serverCharacters[i].Id < serverCharacters[j].Id })

	s.SendPacketMsg(sess.conn, &msgsvr.AccountServersListSuccess{
		Subscription:      account.SubscribedUntil,
		ServersCharacters: serverCharacters,
	})*/

	return nil
}

func (s *session) AccountSetServer(extra string) error {
	/*gameserver, err := s.Login.GameServer(filter.GameServerIdEQ(msg.Id))
	if err != nil {
		return err
	}

	tokenData, err := s.TokenData(sess.AccountId, gameserver.Id, sess.LastAccess, sess.LastIP)
	if err != nil {
		return err
	}

	s.SendPacketMsg(sess.conn, &msgsvr.AccountSelectServerPlainSuccess{
		Host:   gameserver.Host,
		Port:   gameserver.Port,
		Ticket: tokenData,
	})

	s.DeleteSession(sess)*/

	return nil
}
