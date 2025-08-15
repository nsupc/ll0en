package config

import (
	"errors"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	slogbetterstack "github.com/samber/slog-betterstack"
)

type Telegram struct {
	Template string `yaml:"template"`
}

func (t *Telegram) validate() error {
	if t.Template == "" {
		return errors.New("telegram template must be set")
	}

	return nil
}

type Config struct {
	User        string `yaml:"user"`
	Region      string `yaml:"region"`
	MaxRequests int    `yaml:"max_requests"`
	Eurocore    struct {
		BaseUrl  string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"eurocore"`
	Telegrams struct {
		Move   Telegram `yaml:"move"`
		Resign Telegram `yaml:"resign"`
	}
	Log struct {
		Level    string `yaml:"level"`
		Token    string `yaml:"token"`
		Endpoint string `yaml:"endpoint"`
	} `yaml:"log"`
}

func (c *Config) validate() error {
	if c.User == "" {
		return errors.New("user is not set")
	}

	if c.Region == "" {
		return errors.New("region is not set")
	}

	if c.MaxRequests > 0 || c.MaxRequests < 50 {
		c.MaxRequests = 30
	}

	c.Region = strings.ToLower(strings.ReplaceAll(c.Region, " ", "_"))

	if c.Eurocore.BaseUrl == "" || c.Eurocore.Username == "" || c.Eurocore.Password == "" {
		return errors.New("all eurocore parameters must be set")
	}

	err := c.Telegrams.Move.validate()
	if err != nil {
		return err
	}

	err = c.Telegrams.Resign.validate()
	if err != nil {
		return err
	}

	c.Log.Level = strings.ToLower(c.Log.Level)

	if !slices.Contains([]string{"debug", "info", "warn", "error"}, c.Log.Level) {
		c.Log.Level = "info"
	}

	return nil
}

func (c *Config) initLogger() {
	var logger *slog.Logger
	var logLevel slog.Level

	switch c.Log.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	if c.Log.Token != "" && c.Log.Endpoint != "" {
		logger = slog.New(slogbetterstack.Option{
			Token:    c.Log.Token,
			Endpoint: c.Log.Endpoint,
			Level:    logLevel,
		}.NewBetterstackHandler())
	} else {
		logger = slog.Default()
	}

	slog.SetLogLoggerLevel(logLevel)
	slog.SetDefault(logger)
}

// pulls config from ./config.yml or a custom location which can be passed to the
// application as its first cli argument
func Read() (*Config, error) {
	var path string

	if len(os.Args) > 1 {
		path = os.Args[1]
	} else {
		path = "config.yml"
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{}

	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return nil, err
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	config.initLogger()

	return config, nil
}
