package config

import (
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Admin    AdminConfig    `mapstructure:"admin"`
	Paths    PathsConfig    `mapstructure:"paths"`
	System   SystemConfig   `mapstructure:"system"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

type AdminConfig struct {
	Username string `mapstructure:"username"`
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

type PathsConfig struct {
	VhostsDir    string `mapstructure:"vhosts_dir"`
	TemplatesDir string `mapstructure:"templates_dir"`
	SshdConfig   string `mapstructure:"sshd_config"`
	NginxConfDir string `mapstructure:"nginx_conf_dir"`
}

type SystemConfig struct {
	OPanelGroup string `mapstructure:"opanel_group"`
	PHPVersion  string `mapstructure:"php_version"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("database.path", "/var/lib/opanel/opanel.db")
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("admin.username", "admin")
	v.SetDefault("admin.email", "admin@example.com")
	v.SetDefault("admin.password", "changeme")
	v.SetDefault("paths.vhosts_dir", "/var/www/vhosts")
	v.SetDefault("paths.templates_dir", "/opt/opanel/templates")
	v.SetDefault("paths.sshd_config", "/etc/ssh/sshd_config")
	v.SetDefault("paths.nginx_conf_dir", "/etc/nginx/sites-enabled")
	v.SetDefault("system.opanel_group", "opanel_users")
	v.SetDefault("system.php_version", "8.2")

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found; write defaults
		if err := os.MkdirAll(path[:len(path)-len("/config.yaml")], 0755); err == nil {
			_ = v.WriteConfigAs(path)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
