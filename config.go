package main

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var (
	k      = koanf.New(".")
	parser = toml.Parser()

	Config struct {
		Prefixes []string `koanf:"prefixes"`
		Sites    []string `koanf:"sites"`
	}
)

func loadConfig() error {
	if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
		return err
	}

	if err := k.Unmarshal("", &Config); err != nil {
		return err
	}

	return validateConfig()
}

func validateConfig() error {
	for _, prefix := range Config.Prefixes {
		prefixSplit := strings.Split(prefix, "/")
		if len(prefixSplit) != 2 {
			return fmt.Errorf("Config(Prefix4): Prefix not valid: %v", prefix)
		}

		if err := validation.Validate(prefixSplit[0], is.IPv4); err != nil {
			return fmt.Errorf("Config(prefix4): Prefix not valid %w, %v", err, prefix)
		}

		if prefixSplit[1] != "24" {
			return fmt.Errorf("Config(prefix4): Prefix not a /24 %v", prefix)
		}
	}
	return nil
}
