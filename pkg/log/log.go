package log

import (
	"context"
	
	"github.com/go-logr/stdr"
	"github.com/go-logr/logr"
)
// GetLogger returns a stdr.Logger that implements the logr.Logger interface
// and sets the verbosity of the returned logger.
// set v to 0 for info level messages, 
// 1 for debug messages and 2 for trace level message.
// any other verbosity level will default to 0.
func GetLogger(v int) logr.Logger {
	logger := stdr.New(nil).WithName("match")
	// bound check
	if v > 2 || v < 0 {
		v = 0
		logger.Info("Invalid verbosity, setting logger to display info level messages only.")
	}
	stdr.SetVerbosity(v)

	return logger
}

// ContextWithLogger returns a context that has a logr.Logger contained inside,
// which can then be used by Receiver/Send functions in various PSI protocols.
func ContextWithLogger(ctx context.Context, logger logr.Logger) context.Context {
	return logr.NewContext(ctx, logger)
}

// GetLoggerFromContextWithName returns a logr.Logger if it was contained in the context
// otherwise, it returns a fresh logger with verbosity set to 0.
func GetLoggerFromContextWithName(ctx context.Context, name string) logr.Logger {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		logger = GetLogger(0)
	}

	if name != "" {
		return logger.WithName(name)
	}
	return logger
}