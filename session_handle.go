package retrologin

import (
	"context"
	"errors"
	"sort"

	"github.com/kralamoure/dofus"
	"github.com/kralamoure/retro"
	"github.com/kralamoure/retroproto/msgcli"
	"github.com/kralamoure/retroproto/msgsvr"
	prototyp "github.com/kralamoure/retroproto/typ"
)

func (s *session) handleAccountVersion(m msgcli.AccountVersion) error {
	s.version = m
	s.status.Store(statusExpectingAccountCredential)
	return nil
}

func (s *session) handleAccountCredential(m msgcli.AccountCredential) error {
	s.credential = m
	s.status.Store(statusExpectingAccountQueuePosition)
	return nil
}

func (s *session) handleAccountQueuePosition(ctx context.Context) error {
	s.sendMessage(msgsvr.AccountNewQueue{
		Position:    1,
		TotalAbo:    0,
		TotalNonAbo: 1,
		Subscriber:  false,
		QueueId:     0,
	})

	if s.status.Load() == statusExpectingAccountQueuePosition {
		return s.login(ctx)
	}

	return nil
}

func (s *session) handleAccountSearchForFriend(ctx context.Context, m msgcli.AccountSearchForFriend) error {
	user, err := s.svr.dofus.UserByNickname(ctx, m.Pseudo)
	if err != nil {
		if errors.Is(err, dofus.ErrNotFound) {
			s.sendMessage(msgsvr.AccountFriendServerList{})
			return nil
		} else {
			return err
		}
	}

	accounts, err := s.svr.dofus.AccountsByUserId(ctx, user.Id)
	if err != nil {
		return err
	}

	serverIdQty := make(map[int]int)

	for _, account := range accounts {
		characters, err := s.svr.retro.AllCharactersByAccountId(ctx, account.Id)
		if err != nil {
			return err
		}
		for _, character := range characters {
			serverIdQty[character.GameServerId]++
		}
	}

	var serverCharacters []prototyp.AccountServersListServerCharacters

	for serverId, qty := range serverIdQty {
		serverCharacters = append(serverCharacters, prototyp.AccountServersListServerCharacters{
			Id:  serverId,
			Qty: qty,
		})
	}
	sort.Slice(serverCharacters, func(i, j int) bool { return serverCharacters[i].Id < serverCharacters[j].Id })

	s.sendMessage(msgsvr.AccountFriendServerList{ServersCharacters: serverCharacters})

	return nil
}

func (s *session) handleAccountGetServersList(ctx context.Context) error {
	account, err := s.svr.dofus.Account(ctx, s.accountId)
	if err != nil {
		return err
	}

	serverIdQty := make(map[int]int)

	characters, err := s.svr.retro.AllCharactersByAccountId(ctx, s.accountId)
	if err != nil {
		return err
	}
	for _, character := range characters {
		serverIdQty[character.GameServerId]++
	}

	var serverCharacters []prototyp.AccountServersListServerCharacters

	for serverId, qty := range serverIdQty {
		serverCharacters = append(serverCharacters, prototyp.AccountServersListServerCharacters{
			Id:  serverId,
			Qty: qty,
		})
	}
	sort.Slice(serverCharacters, func(i, j int) bool { return serverCharacters[i].Id < serverCharacters[j].Id })

	s.sendMessage(msgsvr.AccountServersListSuccess{
		Subscription:      account.Subscription,
		ServersCharacters: serverCharacters,
	})

	return nil
}

func (s *session) handleAccountSetServer(ctx context.Context, m msgcli.AccountSetServer) error {
	gameServer, err := s.svr.retro.GameServer(ctx, m.Id)
	if err != nil {
		return err
	}

	id, err := s.svr.retro.CreateTicket(ctx, retro.Ticket{
		AccountId:    s.accountId,
		GameServerId: m.Id,
	})
	if err != nil {
		return err
	}

	s.sendMessage(msgsvr.AccountSelectServerPlainSuccess{
		Host:   gameServer.Host,
		Port:   gameServer.Port,
		Ticket: id,
	})

	return errInvalidRequest
}
