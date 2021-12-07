package format

import (
	"math"
	"os"
	"runtime"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

// GetLogger returns a stdr.Logger that implements the logr.Logger interface
// and sets the verbosity of the returned logger.
// set v to 0 for info level messages,
// 1 for debug messages and 2 for trace level message.
// any other verbosity level will default to 0.
func GetLogger(v int) logr.Logger {
	logger := stdr.New(nil)
	// bound check
	if v > 2 || v < 0 {
		v = 0
		logger.Info("Invalid verbosity, setting logger to display info level messages only.")
	}
	stdr.SetVerbosity(v)

	return logger
}

// ShowUsageAndExit displays the usage message to stdout and exit
func ShowUsageAndExit(usage func(), exitcode int) {
	usage()
	os.Exit(exitcode)
}

// MemUsageToStdErr logs the total PSI memory usage, and garbage collector calls
func MemUsageToStdErr(logger logr.Logger) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m) // https://cs.opensource.google/go/go/+/go1.17.1:src/runtime/mstats.go;l=107
	logger.V(1).Info("Final stats", "total memory (GiB)", math.Round(float64(m.Sys)*100/(1024*1024*1024))/100)
	logger.V(1).Info("Final stats", "garbage collector calls", m.NumGC)
}

// ExitOnErr logs the error and exit if error is not nil
func ExitOnErr(logger logr.Logger, err error, msg string) {
	if err != nil {
		logger.Error(err, msg)
		os.Exit(1)
	}
}
