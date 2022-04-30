package retrologin

import (
	"github.com/kralamoure/retroproto/msgcli"
)

type AccountCredential struct {
	msgcli.AccountCredential
}

func (msg AccountCredential) handle(c *client) error {
	c.logger.Infof("handling message %s: %+v", msg.MessageName(), msg.AccountCredential)

	return nil
}
