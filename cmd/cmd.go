package cmd

import (
	"fmt"
	"net/http"

	"github.com/janu-cambrelen/proxy-service/internal/proxyserver"
	"go.uber.org/zap"
)

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

	// new handler with logging middleware
	handler := proxyserver.NewProxyServer(
		cfg.Debug,
		cfg.TargetURL,
		cfg.RequestDelay,
		cfg.BodyMethodsOnly,
		cfg.RejectWith,
		cfg.RejectExact,
		cfg.RejectInsensitive,
		logger,
		proxyserver.RequestCopy{},
	).WithRequestLoggerMiddleware()
	logger.Info("initializing server")
	logger.Debug("server configuration", zap.Any("details", cfg)) // only when DEBUG=true

	// listen and serve handler
	addr := fmt.Sprintf("%s:%v", cfg.Host, cfg.Port)
	if err := http.ListenAndServe(addr, handler); err != http.ErrServerClosed {
		logger.Error("unexpected server error", zap.Error(err))
	}

	return nil
}
