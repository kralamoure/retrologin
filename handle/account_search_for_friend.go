package handle

import (
	"errors"
	"sort"

	"github.com/kralamoure/d1"
	"github.com/kralamoure/d1/filter"
	"github.com/kralamoure/d1/typ"
	"github.com/kralamoure/d1proto/msgcli"
	"github.com/kralamoure/d1proto/msgsvr"
	typ2 "github.com/kralamoure/d1proto/typ"

	"github.com/kralamoure/d1login"
)

func AccountSearchForFriend(s *d1login.Server, sess *d1login.Session, msg msgcli.AccountSearchForFriend) error {
	user, err := s.Login.User(filter.UserNicknameEQ(typ.Nickname(msg.Pseudo)))
	if err != nil {
		if errors.Is(err, d1.ErrResourceNotFound) {
			s.SendPacketMsg(sess.Conn, &msgsvr.AccountFriendServerList{})
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

	s.SendPacketMsg(sess.Conn, &msgsvr.AccountFriendServerList{ServersCharacters: serverCharacters})

	return nil
}
