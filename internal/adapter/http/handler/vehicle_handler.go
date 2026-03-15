package handler

import (
	"strconv"

	"fleet-management/internal/adapter/http/presenter"
	"fleet-management/internal/module/vehicle"

	"github.com/gofiber/fiber/v2"
)

// VehicleHandler handles HTTP requests related to vehicle locations
type VehicleHandler struct {
	svc *vehicle.Service
}

// NewVehicleHandler creates a new VehicleHandler
func NewVehicleHandler(svc *vehicle.Service) *VehicleHandler {
	return &VehicleHandler{svc: svc}
}

// GetLatestLocation godoc
// GET /vehicles/:vehicle_id/location
func (h *VehicleHandler) GetLatestLocation(c *fiber.Ctx) error {
	vehicleID := c.Params("vehicle_id")
	if vehicleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(presenter.ErrorResponse{
			Error: "vehicle_id is required",
		})
	}

	loc, err := h.svc.GetLatestLocation(vehicleID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(presenter.ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.ToLocationResponse(loc))
}

// GetLocationHistory godoc
// GET /vehicles/:vehicle_id/history?start=<unix>&end=<unix>
func (h *VehicleHandler) GetLocationHistory(c *fiber.Ctx) error {
	vehicleID := c.Params("vehicle_id")
	if vehicleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(presenter.ErrorResponse{
			Error: "vehicle_id is required",
		})
	}

	startStr := c.Query("start")
	endStr := c.Query("end")

	if startStr == "" || endStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(presenter.ErrorResponse{
			Error: "query params 'start' and 'end' are required",
		})
	}

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(presenter.ErrorResponse{
			Error: "invalid 'start' timestamp",
		})
	}

	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(presenter.ErrorResponse{
			Error: "invalid 'end' timestamp",
		})
	}

	locations, err := h.svc.GetLocationHistory(vehicleID, start, end)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(presenter.ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.ToHistoryResponse(vehicleID, locations))
}
