package cmd

import (
	"go.uber.org/zap"
)

// NewLogger is a simple wrapper function to return either a
// development or production logger based on the boolean value
// of the DEBUG configuration setting. The development logger writes
// DEBUG and above, whereas the production logger writes INFO and above.
// The developement logger is, among other things, more readable and useful
// for developement; however, it should not be used in production.
func NewLogger(debug bool) (*zap.Logger, error) {
	if debug {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
