package handle

import (
	"sort"

	"github.com/kralamoure/d1/filter"
	"github.com/kralamoure/d1proto/msgcli"
	"github.com/kralamoure/d1proto/msgsvr"
	typ2 "github.com/kralamoure/d1proto/typ"

	"github.com/kralamoure/d1login"
)

func AccountGetServersList(s *d1login.Server, sess *d1login.Session, msg msgcli.AccountGetServersList) error {
	account, err := s.Login.Account(filter.AccountIdEQ(sess.AccountId))
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

	s.SendPacketMsg(sess.Conn, &msgsvr.AccountServersListSuccess{
		Subscription:      account.SubscribedUntil,
		ServersCharacters: serverCharacters,
	})

	return nil
}
