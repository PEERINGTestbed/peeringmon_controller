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

type ConfigPrefix struct {
	Prefix string `koanf:"prefix"`
	ASN    int    `koanf:"asn"`
}
type ConfigSite struct {
	Name     string `koanf:"name"`
	Neighbor string `koanf:"neighbor"`
	ASN      int    `konaf:"asn"`
}

var (
	k      = koanf.New(".")
	parser = toml.Parser()

	Config struct {
		ASN        uint32          `koanf:"asn"`
		RouterID   string          `koanf:"router_id"`
		ListenPort int32           `koanf:"listen_port"`
		Prefixes   []*ConfigPrefix `koanf:"prefixes"`
		Sites      []*ConfigSite   `koanf:"sites"`
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
	for _, configPrefix := range Config.Prefixes {
		prefix := configPrefix.Prefix
		prefixSplit := strings.Split(prefix, "/")
		if len(prefixSplit) != 2 {
			return fmt.Errorf("Config(Prefix4): Prefix not valid: %v", prefix)
		}

		if err := validation.Validate(prefixSplit[0], is.IPv4); err != nil {
			return fmt.Errorf("Config(prefix4): Prefix not valid %w, %v", err, prefix)
		}
	}
	return nil
}
