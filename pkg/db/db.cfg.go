package db

type PGConfig struct {
	Host     string `flag:"db_host" env:"DB_HOST" yaml:"db_host" validate:"required"`
	Port     int    `flag:"db_port" env:"DB_PORT" yaml:"db_port" validate:"min=1,max=65535"`
	Username string `flag:"db_username" env:"DB_USERNAME" yaml:"db_username" validate:"required"`
	Password string `flag:"db_password" env:"DB_PASSWORD" yaml:"db_password" validate:"required"`
	Name     string `flag:"db_name" env:"DB_NAME" yaml:"db_name" validate:"required"`
	SSL      bool   `flag:"db_ssl" env:"DB_SSL" yaml:"db_ssl"`
}
