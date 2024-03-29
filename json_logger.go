/*
Package gomoljson is a JSON logger implementation for gomol.

Message order is not guaranteed during reconnection scenarios.

*/
package gomoljson

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aphistic/gomol"
	"github.com/efritz/backoff"
)

type dialFunc func(network, address string, timeout time.Duration) (net.Conn, error)

type LoggerOption func(cfg *JSONLoggerConfig)

// WithDialTimeout sets the amount of time the logger will try to connect to the host before timing out
func WithDialTimeout(timeout time.Duration) LoggerOption {
	return func(cfg *JSONLoggerConfig) {
		cfg.DialTimeout = timeout
	}
}

// JSONLoggerConfig is the configuration for a JSONLogger
type JSONLoggerConfig struct {
	// netDial allows the function for dialing a net.Conn to be overridden for testing
	netDial dialFunc

	// A URI for the host to connect to in the format: protocol://host:port. Ex: tcp://10.10.10.10:1234
	HostURI string

	// DialTimeout is the amount of time the logger will try to connect to the host before timing out
	DialTimeout time.Duration

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

	// The number of messages to queue during a connection failure before
	// older messages will start being dropped. Defaults to 100
	FailureQueueLength int

	// Whether an active connection is required on initialization. Defaults to false.
	// If true, the Init function will return a nil error on connection failure, but
	// retry to connect in the background.
	AllowDisconnectedInit bool

	// The backoff strategy to use when reconnecting a connection
	ReconnectBackoff backoff.Backoff
}

// JSONLogger is an instance of a JSON logger
type JSONLogger struct {
	base          *gomol.Base
	isInitialized bool
	config        *JSONLoggerConfig

	hostURL *url.URL

	unprefixedMap map[string]bool

	connMut     sync.RWMutex
	conn        net.Conn
	isConnected bool

	failedMut   sync.RWMutex
	failedQueue [][]byte
}

// NewJSONLoggerConfig creates a new configuration with default settings
func NewJSONLoggerConfig(hostURI string) *JSONLoggerConfig {
	return &JSONLoggerConfig{
		netDial: net.DialTimeout,

		HostURI:     hostURI,
		DialTimeout: 60 * time.Second,

		MessageDelimiter: []byte("\n"),

		FieldPrefix:      "",
		UnprefixedFields: []string{},

		LogLevelField:  "level",
		MessageField:   "message",
		TimestampField: "timestamp",

		LogLevelMap: map[gomol.LogLevel]interface{}{
			gomol.LevelDebug:   gomol.LevelDebug.String(),
			gomol.LevelInfo:    gomol.LevelInfo.String(),
			gomol.LevelWarning: gomol.LevelWarning.String(),
			gomol.LevelError:   gomol.LevelError.String(),
			gomol.LevelFatal:   gomol.LevelFatal.String(),
			gomol.LevelNone:    gomol.LevelNone.String(),
		},

		JSONAttrs: map[string]interface{}{},

		FailureQueueLength: 100,

		ReconnectBackoff: backoff.NewExponentialBackoff(
			100*time.Millisecond,
			time.Minute,
		),
	}
}

// NewJSONLogger creates a new logger with the provided configuration
func NewJSONLogger(config *JSONLoggerConfig) (*JSONLogger, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	l := &JSONLogger{
		config:      config,
		failedQueue: make([][]byte, 0),
	}
	return l, nil
}

func (l *JSONLogger) connect() error {
	l.disconnect()

	l.connMut.Lock()
	defer l.connMut.Unlock()

	conn, err := l.config.netDial(l.hostURL.Scheme, l.hostURL.Host, l.config.DialTimeout)
	if err != nil {
		return err
	}
	l.conn = conn
	l.isConnected = true

	return nil
}

func (l *JSONLogger) disconnect() error {
	l.connMut.Lock()
	defer l.connMut.Unlock()

	var err error
	if l.conn != nil {
		err = l.conn.Close()

		// Even if there's an error calling close just drop the
		// reference and consider the connection closed
		l.conn = nil
		l.isConnected = false
	}
	return err
}

// SetBase will set the gomol.Base this logger is associated with
func (l *JSONLogger) SetBase(base *gomol.Base) {
	l.base = base
}

// Healthy will return true if the logger is connected to the remote host
func (l *JSONLogger) Healthy() bool {
	l.connMut.RLock()
	defer l.connMut.RUnlock()

	return l.isConnected
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

	if err := l.connect(); err != nil {
		if !l.config.AllowDisconnectedInit {
			return err
		}

		// Reconnect in background
		l.performReconnect()
	}

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

	// First add base attrs
	if l.base != nil && l.base.BaseAttrs != nil {
		for key, val := range l.base.BaseAttrs.Attrs() {
			if _, ok := l.unprefixedMap[key]; ok {
				msgMap[key] = val
			} else {
				msgMap[l.config.FieldPrefix+key] = val
			}
		}
	}

	if attrs != nil {
		for key, val := range attrs {
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

	if !l.isConnected {
		// If we're not connected then just queue up the message and return
		// an error about not being connected
		l.queueFailure(msgBytes)
		return errors.New("Could not send message")
	}

	err = l.write(msgBytes)
	if err != nil {
		if neterr, ok := err.(net.Error); ok {
			if !neterr.Temporary() {
				// If this isn't a temporary error then queue the message
				// to the failed queue and start reconnecting
				l.queueFailure(msgBytes)
				l.performReconnect()
				return errors.New("Could not send message")
			}
		}
		return err
	}

	return nil
}

func (l *JSONLogger) write(msgBytes []byte) error {
	msgLen := len(msgBytes)
	written := 0
	for {
		l.connMut.RLock()
		if !l.isConnected || l.conn == nil {
			l.connMut.RUnlock()
			return ErrDisconnected
		}

		// Try writing because we at least think we're connected
		n, err := l.conn.Write(msgBytes[written:])
		l.connMut.RUnlock()

		if err != nil {
			return err
		}

		written += n
		if written >= msgLen {
			// Should never be > but check that way just in case so we don't
			// have an infinite loop in a weird edge case
			break
		}
	}

	return nil
}

func (l *JSONLogger) performReconnect() {
	l.disconnect()

	reconnectedChan := make(chan bool)
	go l.tryReconnect(reconnectedChan)
	go l.trySendFailures(reconnectedChan)
}
func (l *JSONLogger) tryReconnect(rcChan chan<- bool) {
	for {
		err := l.connect()
		if err == nil {
			// We're reconnected!
			l.config.ReconnectBackoff.Reset()
			rcChan <- true
			break
		}
		time.Sleep(l.config.ReconnectBackoff.NextInterval())
	}
}
func (l *JSONLogger) trySendFailures(rcChan <-chan bool) {
	<-rcChan

	// Loop through all failed messages and send them all
	for msgBytes := l.dequeueFailure(); msgBytes != nil; msgBytes = l.dequeueFailure() {
		err := l.write(msgBytes)
		if err != nil {
			if neterr, ok := err.(net.Error); ok {
				if !neterr.Temporary() {
					// Not a temporary failure. Queue the message again and start the
					// process all over again
					l.queueFailure(msgBytes)
					l.performReconnect()
					return
				}
			}
		}
	}
}

func (l *JSONLogger) queueFailure(msgBytes []byte) {
	l.failedMut.Lock()
	defer l.failedMut.Unlock()

	if len(l.failedQueue) == l.config.FailureQueueLength {
		l.failedQueue = append(l.failedQueue[1:], msgBytes)
	} else {
		l.failedQueue = append(l.failedQueue, msgBytes)
	}
}

func (l *JSONLogger) failure(idx int) ([]byte, int) {
	l.failedMut.RLock()
	defer l.failedMut.RUnlock()

	if len(l.failedQueue) > idx {
		return l.failedQueue[idx], len(l.failedQueue)
	}

	return nil, len(l.failedQueue)
}

func (l *JSONLogger) failureLen() int {
	l.failedMut.RLock()
	defer l.failedMut.RUnlock()

	return len(l.failedQueue)
}

func (l *JSONLogger) dequeueFailure() []byte {
	l.failedMut.Lock()
	defer l.failedMut.Unlock()

	if len(l.failedQueue) == 0 {
		return nil
	}

	var val []byte
	val, l.failedQueue = l.failedQueue[0], l.failedQueue[1:]
	return val
}
