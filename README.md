gomol-json
============

[![GoDoc](https://godoc.org/github.com/aphistic/gomol-json?status.svg)](https://godoc.org/github.com/aphistic/gomol-json)
[![Build Status](https://img.shields.io/travis/aphistic/gomol-json.svg)](https://travis-ci.org/aphistic/gomol-json)
[![Code Coverage](https://img.shields.io/codecov/c/github/aphistic/gomol-json.svg)](http://codecov.io/github/aphistic/gomol-json?branch=master)

gomol-json is a logger for [gomol](https://github.com/aphistic/gomol) to support logging JSON over a network.

Installation
============

To install gomol-json, simply `go get` it and import it into your project:

    go get github.com/aphistic/gomol-json
    ...
    import "github.com/aphistic/gomol-json"

Examples
========

For brevity a lot of error checking has been omitted, be sure you do your checks!

This is a super basic example of adding a JSON logger to gomol and then logging a few messages:

```go
package main

import (
	"github.com/aphistic/gomol"
)

// Create a new JSON logger, add it to gomol and log a few messages
func Example() {
	// Add a JSON Logger
	jsonCfg := NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	jsonLogger, _ := NewJSONLogger(jsonCfg)
	gomol.AddLogger(jsonLogger)

	// Set some global attrs that will be added to all
	// messages automatically
	gomol.SetAttr("facility", "gomol.example")
	gomol.SetAttr("another_attr", 1234)

	// Initialize the loggers
	gomol.InitLoggers()
	defer gomol.ShutdownLoggers()

	// Log some debug messages with message-level attrs
	// that will be sent only with that message
	for idx := 1; idx <= 10; idx++ {
		gomol.Dbgm(
			gomol.NewAttrs().
				SetAttr("msg_attr1", 4321),
			"Test message %v", idx)
	}
}

```
