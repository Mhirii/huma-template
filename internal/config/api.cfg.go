package config

import (
	"fmt"
	"os"

	cfg "github.com/mhirii/huma-template/pkg/config"
	"github.com/mhirii/huma-template/pkg/db"
	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type APIConfig struct {
	Server ServerConfig
	Logger logging.LoggerConfig
	Auth   AuthConfig
	DB     db.PGConfig
}

var (
	apiConfig APIConfig
	loaded    bool
)

// --- Loader ---
func Load(file_prefix *string) {
	l := logging.L()
	l.Debug().Msg("loading config")
	if loaded {
		return
	}
	loaded = true

	v := viper.New()

	// Bind all config fields recursively
	cfg.BindConfigStruct(v, &apiConfig.Server, "server")
	cfg.BindConfigStruct(v, &apiConfig.Logger, "logger")
	cfg.BindConfigStruct(v, &apiConfig.DB, "db")
	cfg.BindConfigStruct(v, &apiConfig.Auth, "auth")

	configName := "config"
	// Bind CLI flags
	if file_prefix != nil {
		configName = *file_prefix + "-" + configName
	}
	pflag.String(configName+".yaml", "", "Path to config file or directory")
	pflag.Parse()
	v.BindPFlags(pflag.CommandLine)

	// Load ENV variables
	v.AutomaticEnv()

	// Determine config file path from CLI or ENV, default to current directory
	configPath := v.GetString("config")
	if configPath == "" {
		configPath = v.GetString("CONFIG_PATH")
	}
	if configPath == "" {
		configPath = "."
	}
	l.Debug().Msgf("config path %s", configPath)

	if fi, err := os.Stat(configPath); err == nil && !fi.IsDir() {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName(configName)
		v.SetConfigType("yaml")
		v.AddConfigPath(configPath)
	}
	if err := v.ReadInConfig(); err != nil {
		l.Warn().Msgf("failed to read config file: %v", err)
	}

	// Unmarshal to config struct
	if err := v.Unmarshal(&apiConfig); err != nil {
		l.Warn().Msgf("failed to unmarshal config: %v", err)
	}

	// Validate config
	if err := cfg.ValidateConfigStruct(&apiConfig.Server); err != nil {
		panic(fmt.Sprintf("Config for server validation error: %v", err))
	}
	if err := cfg.ValidateConfigStruct(&apiConfig.Logger); err != nil {
		panic(fmt.Sprintf("Config for logger validation error: %v", err))
	}
	if err := cfg.ValidateConfigStruct(&apiConfig.DB); err != nil {
		panic(fmt.Sprintf("Config for DB validation error: %v", err))
	}
	if err := cfg.ValidateConfigStruct(&apiConfig.Auth); err != nil {
		panic(fmt.Sprintf("config AuthConfig validation error: %v", err))
	}
}

// GetAPICfg returns the global config
func GetAPICfg() APIConfig {
	return apiConfig
}
