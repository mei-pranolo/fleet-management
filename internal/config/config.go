package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration loaded from config.yml
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	MQTT     MQTTConfig     `mapstructure:"mqtt"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Geofence GeofenceConfig `mapstructure:"geofence"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            string `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	Name            string `mapstructure:"name"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

type MQTTConfig struct {
	Broker      string `mapstructure:"broker"`
	ClientID    string `mapstructure:"client_id"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	TopicPrefix string `mapstructure:"topic_prefix"`
	QOS         byte   `mapstructure:"qos"`
	KeepAlive   int    `mapstructure:"keep_alive"`
	PingTimeout int    `mapstructure:"ping_timeout"`
}

type RabbitMQConfig struct {
	URL          string `mapstructure:"url"`
	Exchange     string `mapstructure:"exchange"`
	ExchangeType string `mapstructure:"exchange_type"`
	Queue        string `mapstructure:"queue"`
	RoutingKey   string `mapstructure:"routing_key"`
	ConsumerTag  string `mapstructure:"consumer_tag"`
}

type GeofenceConfig struct {
	Points []GeofencePoint `mapstructure:"points"`
}

type GeofencePoint struct {
	Name          string  `mapstructure:"name"`
	Latitude      float64 `mapstructure:"latitude"`
	Longitude     float64 `mapstructure:"longitude"`
	RadiusMeters  float64 `mapstructure:"radius_meters"`
}

// Load reads configuration from config.yml using viper
func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Allow overriding config values via environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// DSN builds the PostgreSQL connection string
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}
