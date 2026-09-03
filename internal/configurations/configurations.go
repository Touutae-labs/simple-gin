// Package configurations loads the runtime config from a YAML file. The
// file path is APP_CONFIG (defaults to ./config.yml). The struct tags
// match the keys in config.yml exactly.
package configurations

import "fmt"

type ServerConfig struct {
	Port             string `koanf:"port"`
	MaxPayloadSizeKB int    `koanf:"max_payload_size_kb"`
	TimeoutSeconds   int    `koanf:"timeout_seconds"`
	BaseURL          string `koanf:"base_url"`
}

type DatabaseConfig struct {
	Host        string `koanf:"host"`
	Port        int    `koanf:"port"`
	User        string `koanf:"user"`
	Password    string `koanf:"password"`
	DBName      string `koanf:"dbname"`
	SSLMode     string `koanf:"sslmode"`
	AutoMigrate bool   `koanf:"auto_migrate"`
}

// DSN returns the libpq-style connection string consumed by both pgx
// (via stdlib) and gorm.io/driver/postgres when DriverName=pgx.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

type Config struct {
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"database"`
}
