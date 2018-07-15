package gomoljson

import (
	"errors"
	"net"
	"time"

	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

type fakeDialer struct {
	DialError   error
	DialErrorOn int

	dialsSinceError int
}

func newFakeDialer() *fakeDialer {
	return &fakeDialer{}
}

func (d *fakeDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	if d.DialError != nil {
		d.dialsSinceError++
		if d.dialsSinceError >= d.DialErrorOn {
			d.dialsSinceError = 0
			return nil, d.DialError
		}
	}

	return newFakeConn(network, address, timeout), nil
}

type fakeBackoff struct {
	NextCalledCount  int
	ResetCalledCount int
}

func (b *fakeBackoff) Reset() {
	b.ResetCalledCount++
}

func (b *fakeBackoff) NextInterval() time.Duration {
	b.NextCalledCount++
	return time.Millisecond * 0
}

type fakeAddr struct {
	AddrNetwork string
	Address     string
}

func (a *fakeAddr) Network() string {
	return a.AddrNetwork
}
func (a *fakeAddr) String() string {
	return a.Address
}

type fakeNetError struct {
	ErrMsg       string
	ErrTimeout   bool
	ErrTemporary bool
}

func newFakeNetError(msg string, timeout bool, temporary bool) *fakeNetError {
	return &fakeNetError{
		ErrMsg:       msg,
		ErrTimeout:   timeout,
		ErrTemporary: temporary,
	}
}

func (e *fakeNetError) Timeout() bool {
	return e.ErrTimeout
}
func (e *fakeNetError) Temporary() bool {
	return e.ErrTemporary
}
func (e *fakeNetError) Error() string {
	return e.ErrMsg
}

type fakeConn struct {
	Written                []byte
	WriteWindowSize        int
	WriteError             error
	WriteErrorOn           int // Number of writes to return the error after
	WriteSuccessAfterError bool

	HasClosed bool

	localAddr  net.Addr
	remoteAddr net.Addr
	timeout    time.Duration

	writesSinceError int
}

func newFakeConn(network, address string, timeout time.Duration) *fakeConn {
	return &fakeConn{
		Written:   make([]byte, 0),
		HasClosed: false,

		localAddr: &fakeAddr{
			AddrNetwork: network,
			Address:     "10.10.10.10:1234",
		},
		remoteAddr: &fakeAddr{
			AddrNetwork: network,
			Address:     address,
		},
		timeout: timeout,
	}
}

func (c *fakeConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (c *fakeConn) Write(b []byte) (n int, err error) {
	if c.WriteError != nil {
		c.writesSinceError++
		if c.writesSinceError >= c.WriteErrorOn {
			c.writesSinceError = 0

			n = 0
			err = c.WriteError
			if c.WriteSuccessAfterError {
				c.WriteError = nil
			}

			return
		}
	}

	if c.WriteWindowSize > 0 {
		c.Written = append(c.Written, b[:c.WriteWindowSize]...)
		return c.WriteWindowSize, nil
	}
	c.Written = append(c.Written, b...)
	return len(b), nil
}

func (c *fakeConn) Close() error {
	c.HasClosed = true
	return nil
}

func (c *fakeConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *fakeConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *fakeConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type MockSuite struct{}

func (s *MockSuite) TestFakeDialerErrorOn(t sweet.T) {
	fd := newFakeDialer()
	fd.DialError = errors.New("Dial error")
	fd.DialErrorOn = 2

	_, err := fd.DialTimeout("tcp", "10.10.10.10:1234", 60*time.Second)
	Expect(err).To(BeNil())

	_, err = fd.DialTimeout("tcp", "10.10.10.10:1234", 60*time.Second)
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("Dial error"))
}

func (s *MockSuite) TestFakeConn(t sweet.T) {
	f := newFakeConn("tcp", "1.2.3.4:4321", 60*time.Second)
	Expect(f.Written).ToNot(BeNil())
	Expect(f.Written).To(HaveLen(0))

	amt, err := f.Write([]byte{0, 1, 2, 3, 4})
	Expect(err).To(BeNil())
	Expect(amt).To(Equal(5))
	Expect(f.Written).To(Equal([]byte{0, 1, 2, 3, 4}))

	Expect(f.localAddr).To(Equal(&fakeAddr{
		AddrNetwork: "tcp",
		Address:     "10.10.10.10:1234",
	}))
	Expect(f.remoteAddr).To(Equal(&fakeAddr{
		AddrNetwork: "tcp",
		Address:     "1.2.3.4:4321",
	}))
}

func (s *MockSuite) TestFakeConnWriteWindowSize(t sweet.T) {
	f := newFakeConn("tcp", "1.2.3.4:4321", 60*time.Second)
	Expect(f.Written).ToNot(BeNil())
	Expect(f.Written).To(HaveLen(0))

	f.WriteWindowSize = 2

	amt, err := f.Write([]byte{0, 1, 2, 3, 4})
	Expect(err).To(BeNil())
	Expect(amt).To(Equal(2))
	Expect(f.Written).To(Equal([]byte{0, 1}))
}

func (s *MockSuite) TestFakeConnWriteError(t sweet.T) {
	f := newFakeConn("tcp", "1.2.3.4:4321", 60*time.Second)
	Expect(f.Written).ToNot(BeNil())
	Expect(f.Written).To(HaveLen(0))

	f.WriteError = errors.New("write error")

	_, err := f.Write([]byte{0, 1, 2, 3, 4})
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("write error"))
}
func (s *MockSuite) TestFakeConnWriteErrorOn(t sweet.T) {
	f := newFakeConn("tcp", "1.2.3.4:4321", 60*time.Second)
	Expect(f.Written).ToNot(BeNil())
	Expect(f.Written).To(HaveLen(0))

	f.WriteError = errors.New("write error")
	f.WriteErrorOn = 2

	_, err := f.Write([]byte{0, 1, 2, 3, 4})
	Expect(err).To(BeNil())

	_, err = f.Write([]byte{0, 1, 2, 3, 4})
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("write error"))
}
func (s *MockSuite) TestFakeConnWriteSuccessAfterError(t sweet.T) {
	f := newFakeConn("tcp", "1.2.3.4:4321", 60*time.Second)
	Expect(f.Written).ToNot(BeNil())
	Expect(f.Written).To(HaveLen(0))

	f.WriteError = errors.New("write error")
	f.WriteSuccessAfterError = true

	_, err := f.Write([]byte{0, 1, 2, 3, 4})
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("write error"))

	_, err = f.Write([]byte{0, 1, 2, 3, 4})
	Expect(err).To(BeNil())
}

func (s *MockSuite) TestFakeBackoff(t sweet.T) {
	b := &fakeBackoff{}

	b.Reset()
	b.Reset()
	Expect(b.ResetCalledCount).To(Equal(2))

	b.NextInterval()
	b.NextInterval()
	b.NextInterval()
	Expect(b.NextCalledCount).To(Equal(3))
}
