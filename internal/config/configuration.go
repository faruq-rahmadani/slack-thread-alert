package config

// Schema is the schema of the configuration file
type Schema struct {
	App    App    `mapstructure:"app"`
	Client Client `mapstructure:"client"`
}

// LoadAll loads configuration from the YAML file of the given environment:
//   - development: config.yaml (project root)
//   - production:  /etc/slack-thread-alert-config/config.yaml
func LoadAll(environment string) (*Schema, error) {
	cfg, err := readYaml(environment)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
