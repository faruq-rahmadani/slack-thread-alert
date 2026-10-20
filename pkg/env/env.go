package env

import "fmt"

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

var env string

// Init stores the current environment, falling back to development when empty
func Init(environment string) error {
	switch environment {
	case "":
		env = EnvDevelopment
	case EnvDevelopment, EnvProduction:
		env = environment
	default:
		return fmt.Errorf("invalid ENV %q, must be one of: %s, %s", environment, EnvDevelopment, EnvProduction)
	}
	return nil
}

func GetEnv() string {
	return env
}

func IsDevelopment() bool {
	return env == EnvDevelopment
}

func IsProduction() bool {
	return env == EnvProduction
}
