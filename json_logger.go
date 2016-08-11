package gomoljson

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/aphistic/gomol"
)

// Allow changing the Dial function that's being called so it can
// be swapped out for testing
var netDial dialFunc = net.Dial

type dialFunc func(network string, address string) (net.Conn, error)

// JSONLoggerConfig is the configuration for a JSONLogger
type JSONLoggerConfig struct {
	// A URI for the host to connect to in the format: protocol://host:port. Ex: tcp://10.10.10.10:1234
	HostURI string

	// The delimiter to use at the end of every message sent. Defaults to '\n'
	MessageDelimiter []byte

	// The prefix to add before every field name in the JSON data. Defaults to a blank string
	FieldPrefix string
	// A list of field names excluded from having the FieldPrefix added
	UnprefixedFields []string

	// The name of the JSON field to put the log level into. Defaults to "level"
	LogLevelField string
	// The name of the JSON field to put the message into. Defaults to "message"
	MessageField string
	// The name of the JSON field to put the timestamp into. Defaults to "timestamp"
	TimestampField string

	// A map to customize the values of each gomol.LogLevel in the JSON message.
	// Defaults to the string value of each gomol.LogLevel
	LogLevelMap map[gomol.LogLevel]interface{}

	// A map of additional attributes to be added to each JSON message sent. This is useful
	// if there fields to send only to a JSON receiver.  These will override any existing
	// attributes already set on a message.
	JSONAttrs map[string]interface{}
}

// JSONLogger is an instance of a JSON logger
type JSONLogger struct {
	base          *gomol.Base
	isInitialized bool
	config        *JSONLoggerConfig

	hostURL *url.URL

	unprefixedMap map[string]bool

	conn net.Conn
}

// NewJSONLoggerConfig creates a new configuration with default settings
func NewJSONLoggerConfig(hostURI string) *JSONLoggerConfig {
	return &JSONLoggerConfig{
		HostURI: hostURI,

		MessageDelimiter: []byte("\n"),

		FieldPrefix:      "",
		UnprefixedFields: make([]string, 0),

		LogLevelField:  "level",
		MessageField:   "message",
		TimestampField: "timestamp",

		LogLevelMap: map[gomol.LogLevel]interface{}{
			gomol.LEVEL_UNKNOWN: gomol.LEVEL_UNKNOWN.String(),
			gomol.LEVEL_DEBUG:   gomol.LEVEL_DEBUG.String(),
			gomol.LEVEL_INFO:    gomol.LEVEL_INFO.String(),
			gomol.LEVEL_WARNING: gomol.LEVEL_WARNING.String(),
			gomol.LEVEL_ERROR:   gomol.LEVEL_ERROR.String(),
			gomol.LEVEL_FATAL:   gomol.LEVEL_FATAL.String(),
			gomol.LEVEL_NONE:    gomol.LEVEL_NONE.String(),
		},

		JSONAttrs: make(map[string]interface{}),
	}
}

// NewJSONLogger creates a new logger with the provided configuration
func NewJSONLogger(config *JSONLoggerConfig) (*JSONLogger, error) {
	l := &JSONLogger{
		config: config,
	}
	return l, nil
}

// SetBase will set the gomol.Base this logger is associated with
func (l *JSONLogger) SetBase(base *gomol.Base) {
	l.base = base
}

// InitLogger does any initialization the logger may need before being used
func (l *JSONLogger) InitLogger() error {
	if len(l.config.HostURI) == 0 {
		return fmt.Errorf("A HostURI must be set")
	}
	hostURL, err := url.Parse(l.config.HostURI)
	if err != nil {
		return fmt.Errorf("Invalid HostURI: %v", err)
	}
	if !strings.Contains(hostURL.Host, ":") {
		return fmt.Errorf("A port must be provided")
	}
	l.hostURL = hostURL

	l.unprefixedMap = make(map[string]bool)
	for _, fieldName := range l.config.UnprefixedFields {
		l.unprefixedMap[fieldName] = true
	}

	conn, err := netDial(l.hostURL.Scheme, l.hostURL.Host)
	if err != nil {
		return err
	}
	l.conn = conn

	l.isInitialized = true
	return nil
}

// IsInitialized returns whether the logger has already been initialized or not
func (l *JSONLogger) IsInitialized() bool {
	return l.isInitialized
}

// ShutdownLogger shuts down the logger and frees any resources that may be used
func (l *JSONLogger) ShutdownLogger() error {
	if l.conn != nil {
		l.conn.Close()
		l.conn = nil
	}
	l.isInitialized = false
	return nil
}

func (l *JSONLogger) marshalJSON(timestamp time.Time, level gomol.LogLevel, attrs map[string]interface{}, msg string) ([]byte, error) {
	msgMap := make(map[string]interface{})
	if attrs != nil {
		for key := range attrs {
			val := attrs[key]
			if _, ok := l.unprefixedMap[key]; ok {
				msgMap[key] = val
			} else {
				msgMap[l.config.FieldPrefix+key] = val
			}
		}
	}

	// Add level
	var levelVal interface{}
	if lval, ok := l.config.LogLevelMap[level]; ok {
		levelVal = lval
	} else {
		return nil, fmt.Errorf("Log level %v does not have a mapping.", level)
	}
	if _, ok := l.unprefixedMap[l.config.LogLevelField]; ok {
		msgMap[l.config.LogLevelField] = levelVal
	} else {
		msgMap[l.config.FieldPrefix+l.config.LogLevelField] = levelVal
	}

	// Add message
	if _, ok := l.unprefixedMap[l.config.MessageField]; ok {
		msgMap[l.config.MessageField] = msg
	} else {
		msgMap[l.config.FieldPrefix+l.config.MessageField] = msg
	}

	// Add timestamp
	if _, ok := l.unprefixedMap[l.config.TimestampField]; ok {
		msgMap[l.config.TimestampField] = timestamp
	} else {
		msgMap[l.config.FieldPrefix+l.config.TimestampField] = timestamp
	}

	// Add any json attrs that are set
	for jsonKey := range l.config.JSONAttrs {
		if jsonVal, ok := l.config.JSONAttrs[jsonKey]; ok {
			if _, ok := l.unprefixedMap[jsonKey]; ok {
				msgMap[jsonKey] = jsonVal
			} else {
				msgMap[l.config.FieldPrefix+jsonKey] = jsonVal
			}
		}
	}

	jsonBytes, err := json.Marshal(msgMap)
	if err != nil {
		return nil, err
	}

	return jsonBytes, nil
}

// Logm sends a JSON log message to the configured host
func (l *JSONLogger) Logm(timestamp time.Time, level gomol.LogLevel, attrs map[string]interface{}, msg string) error {
	if !l.isInitialized {
		return errors.New("JSON logger has not been initialized")
	}

	msgBytes, err := l.marshalJSON(timestamp, level, attrs, msg)
	if err != nil {
		return err
	}
	msgBytes = append(msgBytes, l.config.MessageDelimiter...)

	l.conn.Write(msgBytes)

	return nil
}
