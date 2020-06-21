package handle

import (
	"github.com/kralamoure/d1/filter"
	"github.com/kralamoure/d1proto/msgcli"
	"github.com/kralamoure/d1proto/msgsvr"

	"github.com/kralamoure/d1login"
)

func AccountSetServer(s *d1login.Server, sess *d1login.Session, msg msgcli.AccountSetServer) error {
	gameserver, err := s.Login.GameServer(filter.GameServerIdEQ(msg.Id))
	if err != nil {
		return err
	}

	tokenData, err := s.TokenData(sess.AccountId, gameserver.Id, sess.LastAccess, sess.LastIP)
	if err != nil {
		return err
	}

	s.SendPacketMsg(sess.Conn, &msgsvr.AccountSelectServerPlainSuccess{
		Host:   gameserver.Host,
		Port:   gameserver.Port,
		Ticket: tokenData,
	})

	s.DeleteSession(sess)

	return nil
}
