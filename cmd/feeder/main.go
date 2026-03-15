package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
)

// LocationPayload mirrors the expected MQTT message format
type LocationPayload struct {
	VehicleID string  `json:"vehicle_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

// VehicleConfig holds per-vehicle feeder configuration
type VehicleConfig struct {
	ID            string  `mapstructure:"id"`
	BaseLatitude  float64 `mapstructure:"base_latitude"`
	BaseLongitude float64 `mapstructure:"base_longitude"`
}

// FeederConfig holds the feeder-specific settings
type FeederConfig struct {
	IntervalSeconds int             `mapstructure:"interval_seconds"`
	Vehicles        []VehicleConfig `mapstructure:"vehicles"`
}

// MQTTConfig holds the broker connection settings
type MQTTConfig struct {
	Broker      string `mapstructure:"broker"`
	ClientID    string `mapstructure:"client_id"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	TopicPrefix string `mapstructure:"topic_prefix"`
	QOS         byte   `mapstructure:"qos"`
}

func main() {
	cfg, mqttCfg := loadConfig()

	client := connectMQTT(mqttCfg)
	defer client.Disconnect(250)

	ticker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("[Feeder] Publishing mock location data every %ds for %d vehicles",
		cfg.IntervalSeconds, len(cfg.Vehicles))

	for {
		select {
		case <-ticker.C:
			for _, v := range cfg.Vehicles {
				publishLocation(client, mqttCfg, v)
			}
		case <-quit:
			log.Println("[Feeder] Shutting down...")
			return
		}
	}
}

// publishLocation builds a mock location payload and publishes it to MQTT
func publishLocation(client mqtt.Client, cfg *MQTTConfig, vehicle VehicleConfig) {
	payload := LocationPayload{
		VehicleID: vehicle.ID,
		Latitude:  vehicle.BaseLatitude + jitter(),
		Longitude: vehicle.BaseLongitude + jitter(),
		Timestamp: time.Now().Unix(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Feeder] Failed to marshal payload for %s: %v", vehicle.ID, err)
		return
	}

	topic := fmt.Sprintf("%s/%s/location", cfg.TopicPrefix, vehicle.ID)
	token := client.Publish(topic, cfg.QOS, false, body)
	token.Wait()

	if err := token.Error(); err != nil {
		log.Printf("[Feeder] Failed to publish for %s: %v", vehicle.ID, err)
		return
	}

	log.Printf("[Feeder] Published | vehicle=%s topic=%s lat=%.6f lon=%.6f ts=%d",
		payload.VehicleID, topic, payload.Latitude, payload.Longitude, payload.Timestamp)
}

// connectMQTT creates and connects a MQTT client
func connectMQTT(cfg *MQTTConfig) mqtt.Client {
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.Broker).
		SetClientID(cfg.ClientID).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetOnConnectHandler(func(c mqtt.Client) {
			log.Printf("[Feeder] Connected to MQTT broker: %s", cfg.Broker)
		}).
		SetConnectionLostHandler(func(c mqtt.Client, err error) {
			log.Printf("[Feeder] Connection lost: %v", err)
		})

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username).SetPassword(cfg.Password)
	}

	client := mqtt.NewClient(opts)
	for {
		token := client.Connect()
		token.Wait()
		if token.Error() == nil {
			break
		}
		log.Printf("[Feeder] Failed to connect, retrying in 5s: %v", token.Error())
		time.Sleep(5 * time.Second)
	}

	return client
}

// loadConfig reads feeder_config.yml using viper
func loadConfig() (*FeederConfig, *MQTTConfig) {
	v := viper.New()
	v.SetConfigFile("feeder_config.yml")
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("[Feeder] Failed to load config: %v", err)
	}

	var feeder FeederConfig
	if err := v.UnmarshalKey("feeder", &feeder); err != nil {
		log.Fatalf("[Feeder] Failed to parse feeder config: %v", err)
	}

	var mqttCfg MQTTConfig
	if err := v.UnmarshalKey("mqtt", &mqttCfg); err != nil {
		log.Fatalf("[Feeder] Failed to parse mqtt config: %v", err)
	}

	return &feeder, &mqttCfg
}

// jitter returns a small random coordinate offset to simulate movement
func jitter() float64 {
	return (rand.Float64()*2 - 1) * 0.0005
}
