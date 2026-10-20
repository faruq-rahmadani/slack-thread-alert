package config

import "time"

type App struct {
	Name       string     `mapstructure:"name"`
	HTTPServer HTTPServer `mapstructure:"http_server"`
	Logger     Logger     `mapstructure:"logger"`
}

type HTTPServer struct {
	ListenAddress   string        `mapstructure:"listen_address"`
	Port            string        `mapstructure:"port"`
	GracefulTimeout time.Duration `mapstructure:"graceful_timeout"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
}

type Logger struct {
	Level string `mapstructure:"level"`
	Debug bool   `mapstructure:"debug"`
}
