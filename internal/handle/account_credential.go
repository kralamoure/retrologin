package handle

import (
	"github.com/kralamoure/d1proto/msgcli"

	"github.com/kralamoure/d1login"
)

func AccountCredential(s *d1login.Server, sess *d1login.Session, msg msgcli.AccountCredential) error {
	sess.Credential = msg
	sess.SetStatus(d1login.SessionStatusExpectingFirstQueuePosition)

	return nil
}
