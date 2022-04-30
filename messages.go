package retrologin

import "github.com/kralamoure/retroproto"

type messageIn interface {
	MessageId() retroproto.MsgCliId
	MessageName() string
	Deserialize(extra string) error

	handle(c *client) error
}

type messageOut interface {
	MessageId() retroproto.MsgSvrId
	MessageName() string
	Serialized() (string, error)
}
