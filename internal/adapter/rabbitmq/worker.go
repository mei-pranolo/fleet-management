package rabbitmq

import (
	"log"

	"fleet-management/internal/module/vehicle"
	"fleet-management/internal/infrastructure/messagebroker"
)

// Worker listens to the geofence_alerts queue and logs incoming events
type Worker struct {
	client *messagebroker.RabbitMQClient
}

// NewWorker creates a new RabbitMQ worker
func NewWorker(client *messagebroker.RabbitMQClient) *Worker {
	return &Worker{client: client}
}

// Start registers the consumer and begins processing geofence events
func (w *Worker) Start() error {
	log.Println("[RabbitMQ Worker] Starting geofence alert consumer...")

	return w.client.Consume(func(event *vehicle.GeofenceEvent) {
		log.Printf(
			"[RabbitMQ Worker] Geofence alert | vehicle=%s event=%s lat=%.6f lon=%.6f ts=%d",
			event.VehicleID,
			event.Event,
			event.Location.Latitude,
			event.Location.Longitude,
			event.Timestamp,
		)
	})
}
