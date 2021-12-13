package cmd

import (
	"flag"
	"fmt"
	"net/url"

	"github.com/spf13/viper"
)

// Config defines the server configuration.
type Config struct {
	Source       string // the source from which the config was loaded
	Debug        bool   `mapstructure:"DEBUG"`         // set debug mode
	Host         string `mapstructure:"HOST"`          // proxy server host name
	Port         int    `mapstructure:"PORT"`          // proxy server port number
	TargetURL    string `mapstructure:"TARGET_URL"`    // url of target backend service
	RequestDelay int    `mapstructure:"REQUEST_DELAY"` // number of seconds to delay consecutive requests
}

// validate is method to validate the server configuration.
// add additional validation as needed.
func (c *Config) validate() error {

	// validate TargetURL
	_, err := url.ParseRequestURI(c.TargetURL)
	if err != nil {
		return fmt.Errorf("invalid target url: %s", err.Error())
	}

	return nil
}

// SetConfig loads configuration from a specified file or from flags/defaults.
// It first attempts to set config values using a file that lives at the
// given path and has the given name.  If it encounters an error, it then attempts
// to load using CLI flags or fallback default values. It will return an error if
// both options fail to pass the validation check within the `validate` method on
// the Config struct.
func SetConfig(path string, filename string) (cfg *Config, err error) {
	if cfg, err = loadEnvFile(path, filename); err != nil {

		cfg, err = loadFlags()
		if err != nil {
			return nil, err
		}

	}
	return cfg, nil
}

// loadEnvFile attempts to load the server configuration from a given path and filename.
func loadEnvFile(path string, filename string) (*Config, error) {
	var cfg Config
	viper.AddConfigPath(path)
	viper.SetConfigFile(filename)
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	err = cfg.validate()
	if err != nil {
		return nil, err
	}

	cfg.Source = filename
	return &cfg, nil
}

// loadFlags loads the server configuration from given CLI flags or sets default values.
func loadFlags() (*Config, error) {
	var cfg Config
	flag.BoolVar(
		&cfg.Debug, "debug", false, "set debug mode")
	flag.StringVar(
		&cfg.Host, "host", "", "proxy server host name")
	flag.IntVar(
		&cfg.Port, "port", 8080, "proxy server port number")
	flag.StringVar(
		&cfg.TargetURL, "target-url", "", "url of target backend service")
	flag.IntVar(
		&cfg.RequestDelay, "request-delay", 2, "number of seconds to delay consecutive requests")
	flag.Parse()

	err := cfg.validate()
	if err != nil {
		return nil, err
	}

	cfg.Source = "flags or default values"
	return &cfg, nil
}
