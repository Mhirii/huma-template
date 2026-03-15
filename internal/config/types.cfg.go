package config

type ServerConfig struct {
	Port int `flag:"port" env:"SERVICE_PORT" yaml:"port" default:"8888" validate:"min=1,max=65535"`
}

type AuthConfig struct {
	Secret          string `flag:"auth_secret" env:"AUTH_SECRET" yaml:"auth_secret" validate:"required"`
	AccessTokenTTL  int    `flag:"auth_access_token_ttl" env:"AUTH_ACCESS_TOKEN_TTL" yaml:"auth_access_token_ttl" validate:"min=1,max=86400"`
	RefreshTokenTTL int    `flag:"auth_refresh_token_ttl" env:"AUTH_REFRESH_TOKEN_TTL" yaml:"auth_refresh_token_ttl" validate:"min=1,max=86400"`
	RateLimit       int    `flag:"auth_rate_limit" env:"AUTH_RATE_LIMIT" yaml:"auth_rate_limit" validate:"min=1,max=1000"`
}
