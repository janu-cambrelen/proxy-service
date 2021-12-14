package e2e

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/janu-cambrelen/proxy-service/cmd"
	"github.com/janu-cambrelen/proxy-service/internal/proxyserver"
	"go.uber.org/zap"
)

var tc struct {
	cfg    *cmd.Config
	server *httptest.Server
	client *http.Client
}

func TestMain(m *testing.M) {
	// set config from file or CLI flags
	cfg, err := cmd.SetConfig(".", ".env")
	if err != nil {
		panic(err)
	}

	// new development or production zap logger
	logger, err := cmd.NewLogger(cfg.Debug)
	if err != nil {
		panic(err)
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

	// test config
	tc.cfg = cfg

	// test server
	tc.server = httptest.NewServer(handler)
	defer tc.server.Close()

	// test client
	tc.client = tc.server.Client()

	os.Exit(m.Run())
}
