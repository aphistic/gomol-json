package gomoljson

import (
	gj "."
	"github.com/aphistic/gomol"
)

// Create a new JSON logger, add it to gomol and log a few messages
func Example() {
	// Add a JSON Logger
	jsonCfg := gj.NewJSONLoggerConfig("tcp://10.10.10.10:1234")
	jsonLogger, _ := gj.NewJSONLogger(jsonCfg)
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
