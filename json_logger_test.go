package gomoljson

import (
	"net"
	"testing"
	"time"

	"errors"

	"github.com/aphistic/gomol"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type GomolSuite struct{}

func (s *GomolSuite) SetUpTest(c *C) {
	netDial = fakeDial
}

var _ = Suite(&GomolSuite{})

func (s *GomolSuite) TestDefaultLogLevelMapping(c *C) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	c.Check(cfg.LogLevelMap[gomol.LEVEL_UNKNOWN], Equals, "unknown")
	c.Check(cfg.LogLevelMap[gomol.LEVEL_DEBUG], Equals, "debug")
	c.Check(cfg.LogLevelMap[gomol.LEVEL_INFO], Equals, "info")
	c.Check(cfg.LogLevelMap[gomol.LEVEL_WARNING], Equals, "warn")
	c.Check(cfg.LogLevelMap[gomol.LEVEL_ERROR], Equals, "error")
	c.Check(cfg.LogLevelMap[gomol.LEVEL_FATAL], Equals, "fatal")
	c.Check(cfg.LogLevelMap[gomol.LEVEL_NONE], Equals, "none")
}

func (s *GomolSuite) TestSetBase(c *C) {
	base := gomol.NewBase()

	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, _ := NewJSONLogger(cfg)

	c.Check(l.base, IsNil)
	l.SetBase(base)
	c.Check(l.base, Equals, base)
}

func (s *GomolSuite) TestInitialize(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, _ := NewJSONLogger(cfg)
	c.Assert(l, NotNil)
	err := l.InitLogger()
	c.Assert(err, IsNil)
	c.Check(l.hostURL.Scheme, Equals, "tcp")
	c.Check(l.hostURL.Host, Equals, "1.2.3.4:4321")
}

func (s *GomolSuite) TestInitializeInvalidHostURI(c *C) {
	cfg := NewJSONLoggerConfig("")
	l, _ := NewJSONLogger(cfg)
	c.Assert(l, NotNil)

	err := l.InitLogger()
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "A HostURI must be set")

	cfg.HostURI = ":"
	err = l.InitLogger()
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "Invalid HostURI: parse :: missing protocol scheme")

	cfg.HostURI = "tcp:"
	err = l.InitLogger()
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "A port must be provided")
}

func (s *GomolSuite) TestInitializeConnectFailure(c *C) {
	netDial = fakeDialError

	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, _ := NewJSONLogger(cfg)
	c.Assert(l, NotNil)

	err := l.InitLogger()
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "Dial error")
}

func (s *GomolSuite) TestShutdown(c *C) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, _ := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(l.IsInitialized(), Equals, true)

	conn := l.conn.(*fakeConn)
	err := l.ShutdownLogger()
	c.Assert(err, IsNil)
	c.Check(conn.HasClosed, Equals, true)
}

func (s *GomolSuite) TestMarshalJsonDefault(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LEVEL_ERROR, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	c.Check(err, IsNil)
	c.Check(string(b), Equals, "{"+
		"\"field1\":\"val1\","+
		"\"field2\":2,"+
		"\"level\":\"error\","+
		"\"message\":\"my message\","+
		"\"timestamp\":\"2016-08-10T13:40:13Z\""+
		"}")
}

func (s *GomolSuite) TestMarshalJsonWithJsonAttrs(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.JSONAttrs = map[string]interface{}{
		"json1": 1,
		"json2": "val2",
	}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LEVEL_ERROR, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	c.Check(err, IsNil)
	c.Check(string(b), Equals, "{"+
		"\"field1\":\"val1\","+
		"\"field2\":2,"+
		"\"json1\":1,"+
		"\"json2\":\"val2\","+
		"\"level\":\"error\","+
		"\"message\":\"my message\","+
		"\"timestamp\":\"2016-08-10T13:40:13Z\""+
		"}")
}

func (s *GomolSuite) TestMarshalJsonMarshalError(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	attrs := map[string]interface{}{
		"field1": "val1",
		"field2": 2,
		"invalid": map[int]int{
			1: 2,
			2: 3,
		},
	}
	_, err = l.marshalJSON(ts, gomol.LEVEL_ERROR, attrs, "my message")
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "json: unsupported type: map[int]int")
}

func (s *GomolSuite) TestMarshalJsonFieldPrefix(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FieldPrefix = "_"
	cfg.UnprefixedFields = []string{"field2"}
	cfg.JSONAttrs = map[string]interface{}{
		"json1": 1,
		"json2": "val2",
	}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LEVEL_ERROR, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	c.Check(err, IsNil)
	c.Check(string(b), Equals, "{"+
		"\"_field1\":\"val1\","+
		"\"_json1\":1,"+
		"\"_json2\":\"val2\","+
		"\"_level\":\"error\","+
		"\"_message\":\"my message\","+
		"\"_timestamp\":\"2016-08-10T13:40:13Z\","+
		"\"field2\":2"+
		"}")
}

func (s *GomolSuite) TestMarshalJsonTimestampNoPrefix(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FieldPrefix = "_"
	cfg.UnprefixedFields = []string{cfg.TimestampField}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LEVEL_ERROR, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	c.Check(err, IsNil)
	c.Check(string(b), Equals, "{"+
		"\"_field1\":\"val1\","+
		"\"_field2\":2,"+
		"\"_level\":\"error\","+
		"\"_message\":\"my message\","+
		"\"timestamp\":\"2016-08-10T13:40:13Z\""+
		"}")
}

func (s *GomolSuite) TestMarshalJsonMessageNoPrefix(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FieldPrefix = "_"
	cfg.UnprefixedFields = []string{cfg.MessageField}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LEVEL_ERROR, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	c.Check(err, IsNil)
	c.Check(string(b), Equals, "{"+
		"\"_field1\":\"val1\","+
		"\"_field2\":2,"+
		"\"_level\":\"error\","+
		"\"_timestamp\":\"2016-08-10T13:40:13Z\","+
		"\"message\":\"my message\""+
		"}")
}

func (s *GomolSuite) TestMarshalJsonLevelNoPrefix(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FieldPrefix = "_"
	cfg.UnprefixedFields = []string{cfg.LogLevelField}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LEVEL_ERROR, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	c.Check(err, IsNil)
	c.Check(string(b), Equals, "{"+
		"\"_field1\":\"val1\","+
		"\"_field2\":2,"+
		"\"_message\":\"my message\","+
		"\"_timestamp\":\"2016-08-10T13:40:13Z\","+
		"\"level\":\"error\""+
		"}")
}

func (s *GomolSuite) TestMarshalJsonLevelNoJsonPrefix(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FieldPrefix = "_"
	cfg.UnprefixedFields = []string{"json1"}
	cfg.JSONAttrs = map[string]interface{}{
		"json1": 1,
		"json2": "val2",
	}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LEVEL_ERROR, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	c.Check(err, IsNil)
	c.Check(string(b), Equals, "{"+
		"\"_field1\":\"val1\","+
		"\"_field2\":2,"+
		"\"_json2\":\"val2\","+
		"\"_level\":\"error\","+
		"\"_message\":\"my message\","+
		"\"_timestamp\":\"2016-08-10T13:40:13Z\","+
		"\"json1\":1"+
		"}")
}

func (s *GomolSuite) TestMarshalJsonMissingLevelMapping(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.LogLevelMap = make(map[gomol.LogLevel]interface{})
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	_, err = l.marshalJSON(ts, gomol.LEVEL_ERROR, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "Log level error does not have a mapping.")
}

func (s *GomolSuite) TestMarshalJsonStringLevelMapping(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.LogLevelMap = map[gomol.LogLevel]interface{}{
		gomol.LEVEL_ERROR: "e",
	}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LEVEL_ERROR, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	c.Check(err, IsNil)
	c.Check(string(b), Equals, "{"+
		"\"field1\":\"val1\","+
		"\"field2\":2,"+
		"\"level\":\"e\","+
		"\"message\":\"my message\","+
		"\"timestamp\":\"2016-08-10T13:40:13Z\""+
		"}")
}

func (s *GomolSuite) TestMarshalJsonNumericLevelMapping(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.LogLevelMap = map[gomol.LogLevel]interface{}{
		gomol.LEVEL_ERROR: 1234,
	}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	c.Check(err, IsNil)
	c.Check(l, NotNil)
	c.Check(l.IsInitialized(), Equals, true)

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LEVEL_ERROR, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	c.Check(err, IsNil)
	c.Check(string(b), Equals, "{"+
		"\"field1\":\"val1\","+
		"\"field2\":2,"+
		"\"level\":1234,"+
		"\"message\":\"my message\","+
		"\"timestamp\":\"2016-08-10T13:40:13Z\""+
		"}")
}

func (s *GomolSuite) TestLogmUninitialized(c *C) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, err := NewJSONLogger(cfg)

	err = l.Logm(time.Now(), gomol.LEVEL_DEBUG, nil, "test")
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "JSON logger has not been initialized")
}
func (s *GomolSuite) TestLogmMarshalLevelError(c *C) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	cfg.LogLevelMap = make(map[gomol.LogLevel]interface{})
	l, err := NewJSONLogger(cfg)

	err = l.Logm(time.Now(), gomol.LEVEL_DEBUG, nil, "test")
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "JSON logger has not been initialized")
}
func (s *GomolSuite) TestLogmMarshalJsonError(c *C) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	err = l.Logm(time.Now(), gomol.LEVEL_DEBUG, map[string]interface{}{
		"invalid": map[int]int{
			1: 2,
			2: 3,
		},
	}, "test")
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "json: unsupported type: map[int]int")
}
func (s *GomolSuite) TestLogmWrite(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LEVEL_DEBUG, nil, "test")
	c.Check(err, IsNil)
	conn := l.conn.(*fakeConn)
	c.Check(conn.Written, DeepEquals, []byte("{"+
		"\"level\":\"debug\","+
		"\"message\":\"test\","+
		"\"timestamp\":\"2016-08-11T10:56:33Z\""+
		"}\n"))
}
func (s *GomolSuite) TestLogmWriteCustomDelimiter(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.MessageDelimiter = []byte("DELIMITER")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LEVEL_DEBUG, nil, "test")
	c.Check(err, IsNil)
	conn := l.conn.(*fakeConn)
	c.Check(conn.Written, DeepEquals, []byte("{"+
		"\"level\":\"debug\","+
		"\"message\":\"test\","+
		"\"timestamp\":\"2016-08-11T10:56:33Z\""+
		"}DELIMITER"))
}

// ====================
// Fake connection impl
// ====================

func (s *GomolSuite) TestFakeConn(c *C) {
	f := newFakeConn("tcp", "1.2.3.4:4321")
	c.Assert(f.Written, NotNil)
	c.Check(f.Written, HasLen, 0)

	amt, err := f.Write([]byte{0, 1, 2, 3, 4})
	c.Check(err, IsNil)
	c.Check(amt, Equals, 5)
	c.Check(f.Written, DeepEquals, []byte{0, 1, 2, 3, 4})

	c.Check(f.localAddr, DeepEquals, &fakeAddr{
		AddrNetwork: "tcp",
		Address:     "10.10.10.10:1234",
	})
	c.Check(f.remoteAddr, DeepEquals, &fakeAddr{
		AddrNetwork: "tcp",
		Address:     "1.2.3.4:4321",
	})
}

func fakeDialError(network string, address string) (net.Conn, error) {
	return nil, errors.New("Dial error")
}

func fakeDial(network string, address string) (net.Conn, error) {
	return newFakeConn(network, address), nil
}

type fakeAddr struct {
	AddrNetwork string
	Address     string
}

func (c *fakeAddr) Network() string {
	return c.AddrNetwork
}
func (c *fakeAddr) String() string {
	return c.Address
}

type fakeConn struct {
	Written   []byte
	HasClosed bool

	localAddr  net.Addr
	remoteAddr net.Addr
}

func newFakeConn(network string, address string) *fakeConn {
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
	}
}

func (c *fakeConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (c *fakeConn) Write(b []byte) (n int, err error) {
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
