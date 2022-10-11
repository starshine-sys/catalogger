package bot

import (
	"os"

	"emperror.dev/errors"
	"github.com/BurntSushi/toml"
)

type Config struct {
	Auth AuthConfig `toml:"auth"`
}

type AuthConfig struct {
	Discord  string `toml:"discord"`
	Postgres string `toml:"postgres"`
	Redis    string `toml:"redis"`
}

func ReadConfig(path string) (c Config, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return c, errors.Wrap(err, "read config file")
	}

	err = toml.Unmarshal(b, &c)
	if err != nil {
		return c, errors.Wrap(err, "unmarshal config")
	}
	return c, nil
}
