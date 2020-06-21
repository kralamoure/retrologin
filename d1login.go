package d1login

import (
	"strings"
)

func IsClosedConnError(err error) bool {
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}
