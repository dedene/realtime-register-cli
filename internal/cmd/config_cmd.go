package cmd

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/dedene/realtime-register-cli/internal/config"
	"github.com/dedene/realtime-register-cli/internal/output"
)

// ConfigCmd manages configuration.
type ConfigCmd struct {
	Get  ConfigGetCmd  `cmd:"" help:"Get a config value"`
	Set  ConfigSetCmd  `cmd:"" help:"Set a config value"`
	List ConfigListCmd `cmd:"" help:"List all config values"`
	Path ConfigPathCmd `cmd:"" help:"Show config file path"`
}

// ConfigGetCmd gets a config value.
type ConfigGetCmd struct {
	Key string `arg:"" help:"Config key"`
}

func (c *ConfigGetCmd) Run(_ *RootFlags) error {
	cfg, err := config.ReadConfig()
	if err != nil {
		return &ExitError{Code: CodeError, Err: err}
	}

	var value string
	switch strings.ToLower(c.Key) {
	case "customer":
		value = cfg.Customer
	case "default_tlds":
		value = strings.Join(cfg.DefaultTLDs, ",")
	case "auto_renew":
		if cfg.AutoRenew != nil {
			value = fmt.Sprintf("%t", *cfg.AutoRenew)
		}
	case "keyring_backend":
		value = cfg.KeyringBackend
	default:
		return &ExitError{Code: CodeError, Err: fmt.Errorf("unknown config key: %s", c.Key)}
	}

	fmt.Println(value)
	return nil
}

// ConfigSetCmd sets a config value.
type ConfigSetCmd struct {
	Key   string `arg:"" help:"Config key"`
	Value string `arg:"" help:"Config value"`
}

func (c *ConfigSetCmd) Run(_ *RootFlags) error {
	cfg, err := config.ReadConfig()
	if err != nil {
		return &ExitError{Code: CodeError, Err: err}
	}

	switch strings.ToLower(c.Key) {
	case "customer":
		cfg.Customer = c.Value
	case "default_tlds":
		cfg.DefaultTLDs = strings.Split(c.Value, ",")
	case "auto_renew":
		v := strings.EqualFold(c.Value, "true") || c.Value == "1"
		cfg.AutoRenew = &v
	case "keyring_backend":
		cfg.KeyringBackend = c.Value
	default:
		return &ExitError{Code: CodeError, Err: fmt.Errorf("unknown config key: %s", c.Key)}
	}

	if err := config.WriteConfig(cfg); err != nil {
		return &ExitError{Code: CodeError, Err: err}
	}

	fmt.Printf("Set %s = %s\n", c.Key, c.Value)
	return nil
}

// ConfigListCmd lists all config values.
type ConfigListCmd struct{}

func (c *ConfigListCmd) Run(flags *RootFlags) error {
	cfg, err := config.ReadConfig()
	if err != nil {
		return &ExitError{Code: CodeError, Err: err}
	}

	f := output.NewFormatter(os.Stdout, flags.JSON, flags.Plain, flags.Color == "never")

	if flags.JSON {
		return f.Output(cfg, nil, nil)
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return &ExitError{Code: CodeError, Err: err}
	}
	fmt.Print(string(b))
	return nil
}

// ConfigPathCmd shows config file path.
type ConfigPathCmd struct{}

func (c *ConfigPathCmd) Run(_ *RootFlags) error {
	path, err := config.ConfigPath()
	if err != nil {
		return &ExitError{Code: CodeError, Err: err}
	}
	fmt.Println(path)
	return nil
}
