package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kralamoure/d1proto"
	"github.com/kralamoure/d1proto/msgcli"
	"github.com/kralamoure/d1proto/msgsvr"

	"github.com/kralamoure/d1login"
	"github.com/kralamoure/d1login/handle"
)

func handlePacketData(svr *d1login.Server, sess *d1login.Session, data string) error {
	var id d1proto.MsgCliId
	var extra string
	switch sess.Status() {
	case d1login.SessionStatusExpectingVersion:
		id = "version"
		extra = data
	case d1login.SessionStatusExpectingCredential:
		id = "credential"
		extra = data
	default:
		tmp, ok := d1proto.MsgCliIdByPkt(data)
		if !ok {
			return fmt.Errorf("unknown packet: %q", data)
		}
		id = tmp

		extra = strings.TrimPrefix(data, string(id))
	}

	name, ok := d1proto.MsgCliNameByID(id)
	if !ok {
		name = "Unknown"
	}

	svr.Logger.Debugw("received packet",
		"name", name,
		"data", data,
		"address", sess.Conn.RemoteAddr().String(),
	)

	if !checkFrame(sess.Status(), id) {
		return errors.New("invalid frame")
	}

	switch id {
	case d1proto.AccountVersion:
		msg := msgcli.AccountVersion{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = handle.AccountVersion(svr, sess, msg)
		if err != nil {
			return err
		}
	case d1proto.AccountCredential:
		msg := msgcli.AccountCredential{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = handle.AccountCredential(svr, sess, msg)
		if err != nil {
			return err
		}
	case d1proto.AccountQueuePosition:
		msg := msgcli.AccountQueuePosition{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = handle.AccountQueuePosition(svr, sess, msg)
		if err != nil {
			return err
		}
	case d1proto.AccountGetServersList:
		msg := msgcli.AccountGetServersList{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = handle.AccountGetServersList(svr, sess, msg)
		if err != nil {
			return err
		}
	case d1proto.AccountSearchForFriend:
		msg := msgcli.AccountSearchForFriend{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = handle.AccountSearchForFriend(svr, sess, msg)
		if err != nil {
			return err
		}
	case d1proto.AccountSetServer:
		msg := msgcli.AccountSetServer{}
		err := msg.Deserialize(extra)
		if err != nil {
			return err
		}
		err = handle.AccountSetServer(svr, sess, msg)
		if err != nil {
			return err
		}
	default:
		svr.Logger.Debugw("unhandled packet",
			"name", name,
			"address", sess.Conn.RemoteAddr().String(),
		)
		svr.SendPacketMsg(sess.Conn, &msgsvr.BasicsNoticed{})
	}

	return nil
}

func checkFrame(status d1login.SessionStatus, msgId d1proto.MsgCliId) bool {
	switch status {
	case d1login.SessionStatusExpectingVersion:
		if msgId != d1proto.AccountVersion {
			return false
		}
	case d1login.SessionStatusExpectingCredential:
		if msgId != d1proto.AccountCredential {
			return false
		}
	case d1login.SessionStatusExpectingFirstQueuePosition:
		if msgId != d1proto.AccountQueuePosition {
			return false
		}
	default:
		switch msgId {
		case d1proto.AccountVersion, d1proto.AccountCredential:
			return false
		}
	}

	return true
}
