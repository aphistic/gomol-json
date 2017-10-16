package gomoljson

import (
	"net"
)

var (
	ErrDisconnected = newGomolJsonError("disconnected", false, false)
)

type GomolJsonError struct {
	msg       string
	timeout   bool
	temporary bool
}

var _ net.Error = &GomolJsonError{}

func newGomolJsonError(msg string, timeout bool, temporary bool) *GomolJsonError {
	return &GomolJsonError{
		msg:       msg,
		timeout:   timeout,
		temporary: temporary,
	}
}

func (gje *GomolJsonError) Error() string {
	return gje.msg
}

func (gje *GomolJsonError) Timeout() bool {
	return gje.timeout
}

func (gje *GomolJsonError) Temporary() bool {
	return gje.temporary
}
