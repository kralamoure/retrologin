package handle

import (
	"github.com/kralamoure/d1proto/msgcli"

	"github.com/kralamoure/d1login"
)

func AccountVersion(s *d1login.Server, sess *d1login.Session, msg msgcli.AccountVersion) error {
	sess.Version = msg
	sess.SetStatus(d1login.SessionStatusExpectingCredential)

	return nil
}
