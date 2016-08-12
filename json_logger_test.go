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
	fd := newFakeDialer()
	netDial = fd.Dial
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
	fd := newFakeDialer()
	fd.DialError = errors.New("Dial error")
	netDial = fd.Dial

	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, _ := NewJSONLogger(cfg)
	c.Assert(l, NotNil)

	err := l.InitLogger()
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "Dial error")
}

func (s *GomolSuite) TestConnectWithExistingConnection(c *C) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, _ := NewJSONLogger(cfg)

	l.InitLogger()

	conn := l.conn.(*fakeConn)
	l.connect()
	c.Check(conn, Not(Equals), l.conn)
	c.Check(conn.HasClosed, Equals, true)
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
	c.Assert(err, NotNil)
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
func (s *GomolSuite) TestLogmWritePartial(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	conn := l.conn.(*fakeConn)
	conn.WriteWindowSize = 10

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LEVEL_DEBUG, nil, "test")
	c.Check(err, IsNil)
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
func (s *GomolSuite) TestLogmErrorWhenAlreadyDisconnected(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	l.disconnect()

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LEVEL_DEBUG, nil, "test")
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "Could not send message")

	c.Assert(len(l.failedQueue), Equals, 1)
	c.Check(l.failedQueue[0], DeepEquals, []byte("{"+
		"\"level\":\"debug\","+
		"\"message\":\"test\","+
		"\"timestamp\":\"2016-08-11T10:56:33Z\""+
		"}\n"))
}
func (s *GomolSuite) TestLogmErrorWhenNewlyDisconnected(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	conn := l.conn.(*fakeConn)
	conn.WriteError = newFakeNetError("disconnected", false, false)

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LEVEL_DEBUG, nil, "test")
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "Could not send message")

	c.Assert(len(l.failedQueue), Equals, 1)
	c.Check(l.failedQueue[0], DeepEquals, []byte("{"+
		"\"level\":\"debug\","+
		"\"message\":\"test\","+
		"\"timestamp\":\"2016-08-11T10:56:33Z\""+
		"}\n"))
}
func (s *GomolSuite) TestLogmErrorOnWriteButNotDisconnected(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	conn := l.conn.(*fakeConn)
	conn.WriteError = errors.New("write error")

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LEVEL_DEBUG, nil, "test")
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "write error")
}

func (s *GomolSuite) TestQueueFailure(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FailureQueueLength = 2
	l, _ := NewJSONLogger(cfg)
	l.InitLogger()

	c.Assert(l.failedQueue, NotNil)
	c.Check(len(l.failedQueue), Equals, 0)
	l.queueFailure([]byte{0x00})
	c.Check(len(l.failedQueue), Equals, 1)
	l.queueFailure([]byte{0x01})
	c.Check(len(l.failedQueue), Equals, 2)
	l.queueFailure([]byte{0x02})
	c.Check(len(l.failedQueue), Equals, 2)
	c.Check(l.failedQueue, DeepEquals, [][]byte{{0x01}, {0x02}})
}

func (s *GomolSuite) TestFailureLen(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FailureQueueLength = 2
	l, _ := NewJSONLogger(cfg)
	l.InitLogger()

	c.Check(l.failureLen(), Equals, 0)
	l.queueFailure([]byte{0x00})
	c.Check(l.failureLen(), Equals, 1)
	l.queueFailure([]byte{0x01})
	c.Check(l.failureLen(), Equals, 2)
	l.queueFailure([]byte{0x02})
	c.Check(l.failureLen(), Equals, 2)
}

func (s *GomolSuite) TestDequeueFailure(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, _ := NewJSONLogger(cfg)
	l.InitLogger()

	c.Assert(l.failedQueue, NotNil)

	l.queueFailure([]byte{0x00})
	l.queueFailure([]byte{0x01})
	l.queueFailure([]byte{0x02})

	item := l.dequeueFailure()
	c.Check(item, DeepEquals, []byte{0x00})
	c.Check(len(l.failedQueue), Equals, 2)
	c.Check(l.failedQueue, DeepEquals, [][]byte{{0x01}, {0x02}})

	item = l.dequeueFailure()
	c.Check(item, DeepEquals, []byte{0x01})
	c.Check(len(l.failedQueue), Equals, 1)
	c.Check(l.failedQueue, DeepEquals, [][]byte{{0x02}})

	item = l.dequeueFailure()
	c.Check(item, DeepEquals, []byte{0x02})
	c.Check(len(l.failedQueue), Equals, 0)
	c.Check(l.failedQueue, DeepEquals, [][]byte{})

	item = l.dequeueFailure()
	c.Check(item, IsNil)
}

func (s *GomolSuite) TestTryReconnect(c *C) {
	fd := newFakeDialer()
	fd.DialError = errors.New("dial error")
	fd.DialErrorOn = 2
	netDial = fd.Dial

	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	testBackoff := &fakeBackoff{}
	cfg.ReconnectBackoff = testBackoff
	l, _ := NewJSONLogger(cfg)
	l.InitLogger()

	rcChan := make(chan bool, 1)

	l.tryReconnect(rcChan)

	c.Check(testBackoff.NextCalledCount, Equals, 1)
	c.Check(testBackoff.ResetCalledCount, Equals, 1)

	select {
	case chanRes := <-rcChan:
		c.Check(chanRes, Equals, true)
	default:
		c.Fail()
	}
}

func (s *GomolSuite) TestTrySendFailures(c *C) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, _ := NewJSONLogger(cfg)
	l.InitLogger()

	conn := l.conn.(*fakeConn)
	conn.WriteError = newFakeNetError("net err", false, false)
	conn.WriteErrorOn = 1
	conn.WriteSuccessAfterError = true

	rcChan := make(chan bool, 1)
	rcChan <- true

	l.queueFailure([]byte{0x00})
	l.queueFailure([]byte{0x01})

	l.trySendFailures(rcChan)

	// Super terrible way to do this but it needs to wait for a reconnect
	// to happen before checking the data written
	loops := 0
	for {
		if l.failureLen() == 0 {
			break
		}
		loops++
		if loops > 50 {
			// Failsafe for if things don't work
			break
		}
		time.Sleep(1 * time.Millisecond)
	}

	conn = l.conn.(*fakeConn)
	c.Check(conn.Written, DeepEquals, []byte{0x01, 0x00})
}

// ======================
// Fake impls for testing
// ======================

func (s *GomolSuite) TestFakeDialerErrorOn(c *C) {
	fd := newFakeDialer()
	fd.DialError = errors.New("Dial error")
	fd.DialErrorOn = 2

	_, err := fd.Dial("tcp", "10.10.10.10:1234")
	c.Check(err, IsNil)

	_, err = fd.Dial("tcp", "10.10.10.10:1234")
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "Dial error")
}

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

func (s *GomolSuite) TestFakeConnWriteWindowSize(c *C) {
	f := newFakeConn("tcp", "1.2.3.4:4321")
	c.Assert(f.Written, NotNil)
	c.Check(f.Written, HasLen, 0)

	f.WriteWindowSize = 2

	amt, err := f.Write([]byte{0, 1, 2, 3, 4})
	c.Check(err, IsNil)
	c.Check(amt, Equals, 2)
	c.Check(f.Written, DeepEquals, []byte{0, 1})
}

func (s *GomolSuite) TestFakeConnWriteError(c *C) {
	f := newFakeConn("tcp", "1.2.3.4:4321")
	c.Assert(f.Written, NotNil)
	c.Check(f.Written, HasLen, 0)

	f.WriteError = errors.New("write error")

	_, err := f.Write([]byte{0, 1, 2, 3, 4})
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "write error")
}
func (s *GomolSuite) TestFakeConnWriteErrorOn(c *C) {
	f := newFakeConn("tcp", "1.2.3.4:4321")
	c.Assert(f.Written, NotNil)
	c.Check(f.Written, HasLen, 0)

	f.WriteError = errors.New("write error")
	f.WriteErrorOn = 2

	_, err := f.Write([]byte{0, 1, 2, 3, 4})
	c.Check(err, IsNil)

	_, err = f.Write([]byte{0, 1, 2, 3, 4})
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "write error")
}
func (s *GomolSuite) TestFakeConnWriteSuccessAfterError(c *C) {
	f := newFakeConn("tcp", "1.2.3.4:4321")
	c.Assert(f.Written, NotNil)
	c.Check(f.Written, HasLen, 0)

	f.WriteError = errors.New("write error")
	f.WriteSuccessAfterError = true

	_, err := f.Write([]byte{0, 1, 2, 3, 4})
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "write error")

	_, err = f.Write([]byte{0, 1, 2, 3, 4})
	c.Check(err, IsNil)
}

func (s *GomolSuite) TestFakeBackoff(c *C) {
	b := &fakeBackoff{}

	b.Reset()
	b.Reset()
	c.Check(b.ResetCalledCount, Equals, 2)

	b.NextInterval()
	b.NextInterval()
	b.NextInterval()
	c.Check(b.NextCalledCount, Equals, 3)
}

type fakeDialer struct {
	DialError   error
	DialErrorOn int

	dialsSinceError int
}

func newFakeDialer() *fakeDialer {
	return &fakeDialer{}
}

func (d *fakeDialer) Dial(network string, address string) (net.Conn, error) {
	if d.DialError != nil {
		d.dialsSinceError++
		if d.dialsSinceError >= d.DialErrorOn {
			d.dialsSinceError = 0
			return nil, d.DialError
		}
	}

	return newFakeConn(network, address), nil
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

	writesSinceError int
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
