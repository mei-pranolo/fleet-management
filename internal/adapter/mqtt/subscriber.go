package mqtt

import (
	"encoding/json"
	"fmt"
	"log"

	"fleet-management/internal/config"
	"fleet-management/internal/module/vehicle"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Subscriber holds the MQTT client and delegates location processing to the service
type Subscriber struct {
	client mqtt.Client
	svc    *vehicle.Service
	cfg    *config.MQTTConfig
}

// NewSubscriber creates and connects a new MQTT subscriber
func NewSubscriber(cfg *config.MQTTConfig, svc *vehicle.Service) (*Subscriber, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.Broker).
		SetClientID(cfg.ClientID).
		SetKeepAlive(60).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetOnConnectHandler(func(c mqtt.Client) {
			log.Printf("[MQTT] Connected to broker: %s", cfg.Broker)
		}).
		SetConnectionLostHandler(func(c mqtt.Client, err error) {
			log.Printf("[MQTT] Connection lost: %v", err)
		})

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username).SetPassword(cfg.Password)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	return &Subscriber{client: client, svc: svc, cfg: cfg}, nil
}

// Subscribe starts listening on the wildcard location topic for all vehicles
func (s *Subscriber) Subscribe() error {
	topic := fmt.Sprintf("%s/+/location", s.cfg.TopicPrefix)

	token := s.client.Subscribe(topic, s.cfg.QOS, s.messageHandler)
	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}

	log.Printf("[MQTT] Subscribed to topic: %s", topic)
	return nil
}

// messageHandler parses an incoming MQTT message and forwards it to the service layer
func (s *Subscriber) messageHandler(_ mqtt.Client, msg mqtt.Message) {
	var loc vehicle.Location
	if err := json.Unmarshal(msg.Payload(), &loc); err != nil {
		log.Printf("[MQTT] Failed to parse payload on topic %s: %v", msg.Topic(), err)
		return
	}

	if err := s.svc.ProcessLocation(&loc); err != nil {
		log.Printf("[MQTT] Failed to process location for vehicle %s: %v", loc.VehicleID, err)
		return
	}

	log.Printf("[MQTT] Processed | vehicle=%s lat=%.6f lon=%.6f", loc.VehicleID, loc.Latitude, loc.Longitude)
}

// Disconnect gracefully disconnects from the MQTT broker
func (s *Subscriber) Disconnect() {
	s.client.Disconnect(250)
}
