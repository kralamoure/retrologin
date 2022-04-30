package retrologin

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/kralamoure/retroproto"
	"go.uber.org/zap"
)

var errInvalidRequest = errors.New("invalid request")

type client struct {
	logger *zap.SugaredLogger
	server *Server
	conn   *net.TCPConn
}

func (c *client) String() string {
	return c.conn.RemoteAddr().String()
}

func (c *client) handlePacket(packet string) error {
	id, ok := retroproto.MsgCliIdByPkt(packet)
	if !ok {
		return fmt.Errorf("unknown message ID for packet %q", packet)
	}

	var extra string
	switch id {
	case retroproto.AccountVersion, retroproto.AccountCredential:
		extra = packet
	default:
		extra = strings.TrimPrefix(packet, string(id))
	}

	var msg messageIn

	switch id {
	case retroproto.AccountVersion:
		msg = &AccountVersion{}
	case retroproto.AccountCredential:
		msg = &AccountCredential{}
	default:
		// TODO: Maybe implement an id.Name method somehow? Maybe make it an interface?
		return fmt.Errorf("unhandled packet type with id %q", id)
	}

	if err := msg.Deserialize(extra); err != nil {
		return fmt.Errorf("could not deserialize message %s: %w", msg.MessageName(), err)
	}

	if err := msg.handle(c); err != nil {
		return fmt.Errorf("could not handle message %s: %w", msg.MessageName(), err)
	}

	return nil
}

func (c *client) sendMsg(msg messageOut) error {
	extra, err := msg.Serialized()
	if err != nil {
		return fmt.Errorf("could not serialize message %s: %w", msg.MessageName(), err)
	}

	packet := fmt.Sprint(msg.MessageId(), extra)

	c.logger.Infof("sent message %s: %+v", msg.MessageName(), msg)

	return c.sendPacket(packet)
}

func (c *client) sendPacket(packet string) error {
	_, err := fmt.Fprint(c.conn, packet, "\x00")
	return err
}
