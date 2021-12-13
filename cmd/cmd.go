package cmd

import "go.uber.org/zap"

func Run() error {
	// set config from file or CLI flags
	cfg, err := SetConfig(".", ".env")
	if err != nil {
		return err
	}

	// new development or production zap logger
	logger, err := NewLogger(cfg.Debug)
	if err != nil {
		return err
	}
	defer logger.Sync()

	logger.Debug("server configuration", zap.Any("config", cfg)) // only when DEBUG=true

	return nil
}
