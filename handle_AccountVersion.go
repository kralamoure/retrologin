package retrologin

import (
	"github.com/kralamoure/retroproto/msgcli"
)

type AccountVersion struct {
	msgcli.AccountVersion
}

func (msg AccountVersion) handle(c *client) error {
	c.logger.Infof("handling message %s: %+v", msg.MessageName(), msg.AccountVersion)

	return nil
}
