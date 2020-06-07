package d1login

import (
	"strings"

	"github.com/kralamoure/d1/service/login"
	"go.uber.org/zap"
)

type Config struct {
	Login  login.Service
	Logger *zap.SugaredLogger
	// SharedKey should be 32 bytes long
	SharedKey []byte
}

func IsClosedConnError(err error) bool {
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}
