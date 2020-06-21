package d1login

func (s *Session) handleAccountVersion(extra string) error {
	/*sess.Version = msg
	sess.SetStatus(d1login.SessionStatusExpectingCredential)*/
	s.svr.logger.Info(extra)

	return nil
}

func (s *Session) handleAccountCredential(extra string) error {
	/*sess.Credential = msg
	sess.SetStatus(d1login.SessionStatusExpectingFirstQueuePosition)*/

	return nil
}

func (s *Session) handleAccountQueuePosition(extra string) error {
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

	return nil
}

func (s *Session) handleAccountSearchForFriend(extra string) error {
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

func (s *Session) AccountGetServersList(extra string) error {
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

func (s *Session) AccountSetServer(extra string) error {
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
