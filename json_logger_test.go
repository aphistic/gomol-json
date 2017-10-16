package gomoljson

import (
	"errors"
	"testing"
	"time"

	"github.com/aphistic/gomol"
	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	RegisterFailHandler(sweet.GomegaFail)

	sweet.Run(m, func(s *sweet.S) {
		s.AddSuite(&GomolSuite{})
		s.AddSuite(&MockSuite{})
	})
}

type GomolSuite struct{}

func (s *GomolSuite) SetUpTest(t sweet.T) {
	fd := newFakeDialer()
	netDial = fd.Dial
}

func (s *GomolSuite) TestDefaultLogLevelMapping(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	Expect(cfg.LogLevelMap[gomol.LevelDebug]).To(Equal("debug"))
	Expect(cfg.LogLevelMap[gomol.LevelInfo]).To(Equal("info"))
	Expect(cfg.LogLevelMap[gomol.LevelWarning]).To(Equal("warn"))
	Expect(cfg.LogLevelMap[gomol.LevelError]).To(Equal("error"))
	Expect(cfg.LogLevelMap[gomol.LevelFatal]).To(Equal("fatal"))
	Expect(cfg.LogLevelMap[gomol.LevelNone]).To(Equal("none"))
}

func (s *GomolSuite) TestSetBase(t sweet.T) {
	base := gomol.NewBase()

	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, _ := NewJSONLogger(cfg)

	Expect(l.base).To(BeNil())
	l.SetBase(base)
	Expect(l.base).To(Equal(base))
}

func (s *GomolSuite) TestInitialize(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, _ := NewJSONLogger(cfg)
	Expect(l).ToNot(BeNil())

	err := l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l.hostURL.Scheme).To(Equal("tcp"))
	Expect(l.hostURL.Host).To(Equal("1.2.3.4:4321"))
}

func (s *GomolSuite) TestInitializeInvalidHostURI(t sweet.T) {
	cfg := NewJSONLoggerConfig("")
	l, _ := NewJSONLogger(cfg)
	Expect(l).ToNot(BeNil())

	err := l.InitLogger()
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("A HostURI must be set"))

	cfg.HostURI = ":"
	err = l.InitLogger()
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("Invalid HostURI: parse :: missing protocol scheme"))

	cfg.HostURI = "tcp:"
	err = l.InitLogger()
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("A port must be provided"))
}

func (s *GomolSuite) TestInitializeConnectFailure(t sweet.T) {
	fd := newFakeDialer()
	fd.DialError = errors.New("Dial error")
	netDial = fd.Dial

	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, _ := NewJSONLogger(cfg)
	Expect(l).ToNot(BeNil())

	err := l.InitLogger()
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("Dial error"))
}

func (s *GomolSuite) TestConnectWithExistingConnection(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, _ := NewJSONLogger(cfg)

	l.InitLogger()

	conn := l.conn.(*fakeConn)
	l.connect()
	Expect(conn).ToNot(Equal(l.conn))
	Expect(conn.HasClosed).To(BeTrue())
}

func (s *GomolSuite) TestShutdown(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, _ := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(l.IsInitialized()).To(BeTrue())

	conn := l.conn.(*fakeConn)
	err := l.ShutdownLogger()
	Expect(err).To(BeNil())
	Expect(conn.HasClosed).To(BeTrue())
}

func (s *GomolSuite) TestMarshalJsonDefault(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LevelError, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	Expect(err).To(BeNil())
	Expect(string(b)).To(Equal(
		"{" +
			"\"field1\":\"val1\"," +
			"\"field2\":2," +
			"\"level\":\"error\"," +
			"\"message\":\"my message\"," +
			"\"timestamp\":\"2016-08-10T13:40:13Z\"" +
			"}",
	))
}

func (s *GomolSuite) TestMarshalJsonWithJsonAttrs(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.JSONAttrs = map[string]interface{}{
		"json1": 1,
		"json2": "val2",
	}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LevelError, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	Expect(err).To(BeNil())
	Expect(string(b)).To(Equal(
		"{" +
			"\"field1\":\"val1\"," +
			"\"field2\":2," +
			"\"json1\":1," +
			"\"json2\":\"val2\"," +
			"\"level\":\"error\"," +
			"\"message\":\"my message\"," +
			"\"timestamp\":\"2016-08-10T13:40:13Z\"" +
			"}",
	))
}

func (s *GomolSuite) TestMarshalJsonWithBaseAttrs(t sweet.T) {
	b := gomol.NewBase()
	b.SetAttr("base1", 1)
	b.SetAttr("base2", "val2")

	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.JSONAttrs = map[string]interface{}{
		"json1": 1,
		"json2": "val2",
	}

	l, err := NewJSONLogger(cfg)
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())

	err = b.AddLogger(l)
	Expect(err).To(BeNil())

	err = b.InitLoggers()
	Expect(err).To(BeNil())
	Expect(b.IsInitialized()).To(BeTrue())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	data, err := l.marshalJSON(ts, gomol.LevelError, nil, "my message")
	Expect(err).To(BeNil())
	Expect(string(data)).To(Equal(
		"{" +
			"\"base1\":1," +
			"\"base2\":\"val2\"," +
			"\"json1\":1," +
			"\"json2\":\"val2\"," +
			"\"level\":\"error\"," +
			"\"message\":\"my message\"," +
			"\"timestamp\":\"2016-08-10T13:40:13Z\"" +
			"}",
	))
}

func (s *GomolSuite) TestMarshalJsonMarshalError(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	attrs := map[string]interface{}{
		"field1":  "val1",
		"field2":  2,
		"invalid": func() string { return "i'm a function!" },
	}
	_, err = l.marshalJSON(ts, gomol.LevelError, attrs, "my message")
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("json: unsupported type: func() string"))
}

func (s *GomolSuite) TestMarshalJsonFieldPrefix(t sweet.T) {
	b := gomol.NewBase()
	b.SetAttr("base1", 1)
	b.SetAttr("base2", "val2")

	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FieldPrefix = "_"
	cfg.UnprefixedFields = []string{"base2", "json2", "field2"}
	cfg.JSONAttrs = map[string]interface{}{
		"json1": 1,
		"json2": "val2",
	}
	l, err := NewJSONLogger(cfg)
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())

	b.AddLogger(l)

	err = b.InitLoggers()
	Expect(err).To(BeNil())
	Expect(b.IsInitialized()).To(BeTrue())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	data, err := l.marshalJSON(ts, gomol.LevelError, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	Expect(err).To(BeNil())
	Expect(string(data)).To(Equal(
		"{" +
			"\"_base1\":1," +
			"\"_field1\":\"val1\"," +
			"\"_json1\":1," +
			"\"_level\":\"error\"," +
			"\"_message\":\"my message\"," +
			"\"_timestamp\":\"2016-08-10T13:40:13Z\"," +
			"\"base2\":\"val2\"," +
			"\"field2\":2," +
			"\"json2\":\"val2\"" +
			"}",
	))
}

func (s *GomolSuite) TestMarshalJsonTimestampNoPrefix(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FieldPrefix = "_"
	cfg.UnprefixedFields = []string{cfg.TimestampField}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LevelError, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	Expect(err).To(BeNil())
	Expect(string(b)).To(Equal(
		"{" +
			"\"_field1\":\"val1\"," +
			"\"_field2\":2," +
			"\"_level\":\"error\"," +
			"\"_message\":\"my message\"," +
			"\"timestamp\":\"2016-08-10T13:40:13Z\"" +
			"}",
	))
}

func (s *GomolSuite) TestMarshalJsonMessageNoPrefix(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FieldPrefix = "_"
	cfg.UnprefixedFields = []string{cfg.MessageField}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LevelError, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	Expect(err).To(BeNil())
	Expect(string(b)).To(Equal(
		"{" +
			"\"_field1\":\"val1\"," +
			"\"_field2\":2," +
			"\"_level\":\"error\"," +
			"\"_timestamp\":\"2016-08-10T13:40:13Z\"," +
			"\"message\":\"my message\"" +
			"}",
	))
}

func (s *GomolSuite) TestMarshalJsonLevelNoPrefix(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FieldPrefix = "_"
	cfg.UnprefixedFields = []string{cfg.LogLevelField}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LevelError, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	Expect(err).To(BeNil())
	Expect(string(b)).To(Equal(
		"{" +
			"\"_field1\":\"val1\"," +
			"\"_field2\":2," +
			"\"_message\":\"my message\"," +
			"\"_timestamp\":\"2016-08-10T13:40:13Z\"," +
			"\"level\":\"error\"" +
			"}",
	))
}

func (s *GomolSuite) TestMarshalJsonLevelNoJsonPrefix(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FieldPrefix = "_"
	cfg.UnprefixedFields = []string{"json1"}
	cfg.JSONAttrs = map[string]interface{}{
		"json1": 1,
		"json2": "val2",
	}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LevelError, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	Expect(err).To(BeNil())
	Expect(string(b)).To(Equal(
		"{" +
			"\"_field1\":\"val1\"," +
			"\"_field2\":2," +
			"\"_json2\":\"val2\"," +
			"\"_level\":\"error\"," +
			"\"_message\":\"my message\"," +
			"\"_timestamp\":\"2016-08-10T13:40:13Z\"," +
			"\"json1\":1" +
			"}",
	))
}

func (s *GomolSuite) TestMarshalJsonMissingLevelMapping(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.LogLevelMap = make(map[gomol.LogLevel]interface{})
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	_, err = l.marshalJSON(ts, gomol.LevelError, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("Log level error does not have a mapping."))
}

func (s *GomolSuite) TestMarshalJsonStringLevelMapping(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.LogLevelMap = map[gomol.LogLevel]interface{}{
		gomol.LevelError: "e",
	}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LevelError, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	Expect(err).To(BeNil())
	Expect(string(b)).To(Equal(
		"{" +
			"\"field1\":\"val1\"," +
			"\"field2\":2," +
			"\"level\":\"e\"," +
			"\"message\":\"my message\"," +
			"\"timestamp\":\"2016-08-10T13:40:13Z\"" +
			"}",
	))
}

func (s *GomolSuite) TestMarshalJsonNumericLevelMapping(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.LogLevelMap = map[gomol.LogLevel]interface{}{
		gomol.LevelError: 1234,
	}
	l, err := NewJSONLogger(cfg)
	l.InitLogger()
	Expect(err).To(BeNil())
	Expect(l).ToNot(BeNil())
	Expect(l.IsInitialized()).To(BeTrue())

	ts := time.Date(2016, 8, 10, 13, 40, 13, 0, time.UTC)
	b, err := l.marshalJSON(ts, gomol.LevelError, map[string]interface{}{
		"field1": "val1",
		"field2": 2,
	}, "my message")
	Expect(err).To(BeNil())
	Expect(string(b)).To(Equal(
		"{" +
			"\"field1\":\"val1\"," +
			"\"field2\":2," +
			"\"level\":1234," +
			"\"message\":\"my message\"," +
			"\"timestamp\":\"2016-08-10T13:40:13Z\"" +
			"}",
	))
}

func (s *GomolSuite) TestLogmUninitialized(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, err := NewJSONLogger(cfg)

	err = l.Logm(time.Now(), gomol.LevelDebug, nil, "test")
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("JSON logger has not been initialized"))
}
func (s *GomolSuite) TestLogmMarshalLevelError(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	cfg.LogLevelMap = make(map[gomol.LogLevel]interface{})
	l, err := NewJSONLogger(cfg)

	err = l.Logm(time.Now(), gomol.LevelDebug, nil, "test")
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("JSON logger has not been initialized"))
}
func (s *GomolSuite) TestLogmMarshalJsonError(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	err = l.Logm(time.Now(), gomol.LevelDebug, map[string]interface{}{
		"invalid": func() string { return "i'm a function!" },
	}, "test")
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("json: unsupported type: func() string"))
}
func (s *GomolSuite) TestLogmWrite(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LevelDebug, nil, "test")
	Expect(err).To(BeNil())
	conn := l.conn.(*fakeConn)
	Expect(conn.Written).To(Equal([]byte(
		"{" +
			"\"level\":\"debug\"," +
			"\"message\":\"test\"," +
			"\"timestamp\":\"2016-08-11T10:56:33Z\"" +
			"}\n",
	)))
}
func (s *GomolSuite) TestLogmWritePartial(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	conn := l.conn.(*fakeConn)
	conn.WriteWindowSize = 10

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LevelDebug, nil, "test")
	Expect(err).To(BeNil())
	Expect(conn.Written).To(Equal([]byte(
		"{" +
			"\"level\":\"debug\"," +
			"\"message\":\"test\"," +
			"\"timestamp\":\"2016-08-11T10:56:33Z\"" +
			"}\n",
	)))
}
func (s *GomolSuite) TestLogmWriteCustomDelimiter(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.MessageDelimiter = []byte("DELIMITER")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LevelDebug, nil, "test")
	Expect(err).To(BeNil())
	conn := l.conn.(*fakeConn)
	Expect(conn.Written).To(Equal([]byte(
		"{" +
			"\"level\":\"debug\"," +
			"\"message\":\"test\"," +
			"\"timestamp\":\"2016-08-11T10:56:33Z\"" +
			"}DELIMITER",
	)))
}
func (s *GomolSuite) TestLogmErrorWhenAlreadyDisconnected(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	l.disconnect()

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LevelDebug, nil, "test")
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("Could not send message"))

	Expect(len(l.failedQueue)).To(Equal(1))
	Expect(l.failedQueue[0]).To(Equal([]byte(
		"{" +
			"\"level\":\"debug\"," +
			"\"message\":\"test\"," +
			"\"timestamp\":\"2016-08-11T10:56:33Z\"" +
			"}\n",
	)))
}
func (s *GomolSuite) TestLogmErrorWhenNewlyDisconnected(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	conn := l.conn.(*fakeConn)
	conn.WriteError = newFakeNetError("disconnected", false, false)

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LevelDebug, nil, "test")
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("Could not send message"))

	Expect(len(l.failedQueue)).To(Equal(1))
	Expect(l.failedQueue[0]).To(Equal([]byte(
		"{" +
			"\"level\":\"debug\"," +
			"\"message\":\"test\"," +
			"\"timestamp\":\"2016-08-11T10:56:33Z\"" +
			"}\n",
	)))
}
func (s *GomolSuite) TestLogmErrorOnWriteButNotDisconnected(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, err := NewJSONLogger(cfg)
	l.InitLogger()

	conn := l.conn.(*fakeConn)
	conn.WriteError = errors.New("write error")

	ts := time.Date(2016, 8, 11, 10, 56, 33, 0, time.UTC)
	err = l.Logm(ts, gomol.LevelDebug, nil, "test")
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal("write error"))
}

func (s *GomolSuite) TestQueueFailure(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FailureQueueLength = 2
	l, _ := NewJSONLogger(cfg)
	l.InitLogger()

	Expect(l.failedQueue).ToNot(BeNil())
	Expect(len(l.failedQueue)).To(Equal(0))
	l.queueFailure([]byte{0x00})
	Expect(len(l.failedQueue)).To(Equal(1))
	l.queueFailure([]byte{0x01})
	Expect(len(l.failedQueue)).To(Equal(2))
	l.queueFailure([]byte{0x02})
	Expect(len(l.failedQueue)).To(Equal(2))
	Expect(l.failedQueue).To(Equal([][]byte{{0x01}, {0x02}}))
}

func (s *GomolSuite) TestFailureLen(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	cfg.FailureQueueLength = 2
	l, _ := NewJSONLogger(cfg)
	l.InitLogger()

	Expect(l.failureLen()).To(Equal(0))
	l.queueFailure([]byte{0x00})
	Expect(l.failureLen()).To(Equal(1))
	l.queueFailure([]byte{0x01})
	Expect(l.failureLen()).To(Equal(2))
	l.queueFailure([]byte{0x02})
	Expect(l.failureLen()).To(Equal(2))
}

func (s *GomolSuite) TestDequeueFailure(t sweet.T) {
	cfg := NewJSONLoggerConfig("tcp://1.2.3.4:4321")
	l, _ := NewJSONLogger(cfg)
	l.InitLogger()

	Expect(l.failedQueue).ToNot(BeNil())

	l.queueFailure([]byte{0x00})
	l.queueFailure([]byte{0x01})
	l.queueFailure([]byte{0x02})

	item := l.dequeueFailure()
	Expect(item).To(Equal([]byte{0x00}))
	Expect(len(l.failedQueue)).To(Equal(2))
	Expect(l.failedQueue).To(Equal([][]byte{{0x01}, {0x02}}))

	item = l.dequeueFailure()
	Expect(item).To(Equal([]byte{0x01}))
	Expect(len(l.failedQueue)).To(Equal(1))
	Expect(l.failedQueue).To(Equal([][]byte{{0x02}}))

	item = l.dequeueFailure()
	Expect(item).To(Equal([]byte{0x02}))
	Expect(len(l.failedQueue)).To(Equal(0))
	Expect(l.failedQueue).To(Equal([][]byte{}))

	item = l.dequeueFailure()
	Expect(item).To(BeNil())
}

func (s *GomolSuite) TestTryReconnect(t sweet.T) {
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

	Expect(testBackoff.NextCalledCount).To(Equal(1))
	Expect(testBackoff.ResetCalledCount).To(Equal(1))

	Eventually(rcChan).Should(Receive(Equal(true)))
}

func (s *GomolSuite) TestTrySendFailures(t sweet.T) {
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
	Expect(conn.Written).To(Equal([]byte{0x01, 0x00}))
}
