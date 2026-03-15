package route

import (
	"fleet-management/internal/adapter/http/handler"

	"github.com/gofiber/fiber/v2"
)

// Register wires all HTTP routes to their handlers
func Register(app *fiber.App, vehicleHandler *handler.VehicleHandler) {
	v1 := app.Group("/vehicles")

	// GET /vehicles/:vehicle_id/location
	v1.Get("/:vehicle_id/location", vehicleHandler.GetLatestLocation)

	// GET /vehicles/:vehicle_id/history?start=&end=
	v1.Get("/:vehicle_id/history", vehicleHandler.GetLocationHistory)
}
