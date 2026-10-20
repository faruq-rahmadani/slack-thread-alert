package config

import (
	"fmt"
	"log"

	"github.com/faruqrahmadani/slack-thread-alert/pkg/env"
	"github.com/spf13/viper"
)

// configFiles maps each environment to its hardcoded config file location
var configFiles = map[string]string{
	env.EnvDevelopment: "config.yaml", // project root
	env.EnvProduction:  "/etc/slack-thread-alert-config/config.yaml",
}

// readYaml reads the configuration from the config file of the given environment
func readYaml(environment string) (*Schema, error) {
	configFile, ok := configFiles[environment]
	if !ok {
		return nil, fmt.Errorf("no config file for environment %q", environment)
	}

	log.Printf("Reading configuration for environment %s: %s", environment, configFile)

	v := viper.New()
	v.SetConfigFile(configFile)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var schema Schema
	if err := v.Unmarshal(&schema); err != nil {
		return nil, err
	}

	return &schema, nil
}
