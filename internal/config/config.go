package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	MariaDB  MariaDBConfig  `mapstructure:"mariadb"`
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
	Secret      string `mapstructure:"secret"`
	ExpiryHours int    `mapstructure:"expiry_hours"`
}

type MariaDBConfig struct {
	SocketPath string `mapstructure:"socket_path"`
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
}

type AdminConfig struct {
	Username string `mapstructure:"username"`
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

type PathsConfig struct {
	VhostsDir       string `mapstructure:"vhosts_dir"`
	TemplatesDir    string `mapstructure:"templates_dir"`
	SshdConfig      string `mapstructure:"sshd_config"`
	NginxConfDir    string `mapstructure:"nginx_conf_dir"`
	PHPFPMPoolDir   string `mapstructure:"php_fpm_pool_dir"`
	PHPFPMSocketDir string `mapstructure:"php_fpm_socket_dir"`
}

type SystemConfig struct {
	OPanelGroup string `mapstructure:"opanel_group"`
	PHPVersion  string `mapstructure:"php_version"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8443)
	v.SetDefault("database.path", "/opt/opanel/db/opanel.db")
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.expiry_hours", 24)
	v.SetDefault("admin.username", "admin")
	v.SetDefault("admin.email", "admin@localhost")
	v.SetDefault("admin.password", "admin")
	v.SetDefault("paths.vhosts_dir", "/var/www/vhosts")
	v.SetDefault("paths.templates_dir", "/opt/opanel/templates")
	v.SetDefault("paths.sshd_config", "/etc/ssh/sshd_config")
	v.SetDefault("paths.nginx_conf_dir", "/etc/nginx/sites-enabled")
	v.SetDefault("mariadb.socket_path", "/var/run/mysqld/mysqld.sock")
	v.SetDefault("mariadb.host", "localhost")
	v.SetDefault("mariadb.port", 3306)
	v.SetDefault("paths.php_fpm_pool_dir", "/etc/php/8.4/fpm/pool.d")
	v.SetDefault("paths.php_fpm_socket_dir", "/run/php")
	v.SetDefault("system.opanel_group", "opanel_users")
	v.SetDefault("system.php_version", "8.4")

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found; write defaults
		if err := os.MkdirAll(filepath.Dir(path), 0755); err == nil {
			_ = v.WriteConfigAs(path)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
