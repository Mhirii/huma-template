package logging

type LoggerConfig struct {
	LogLevel  string `flag:"log_level" env:"LOG_LEVEL" yaml:"log_level" default:"info" validate:"oneof=debug info warn error"`
	LogFormat string `flag:"log_format" env:"LOG_FORMAT" yaml:"log_format" default:"text" validate:"oneof=text json"`
	LogFile   string `flag:"log_file" env:"LOG_FILE" yaml:"log_file" default:"app.log"`
}
