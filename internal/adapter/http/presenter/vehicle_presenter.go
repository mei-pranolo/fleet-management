package presenter

import "fleet-management/internal/module/vehicle"

// LocationResponse is the API response shape for a single vehicle location
type LocationResponse struct {
	VehicleID string  `json:"vehicle_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

// HistoryResponse is the API response shape for a slice of vehicle locations
type HistoryResponse struct {
	VehicleID string              `json:"vehicle_id"`
	Total     int                 `json:"total"`
	Locations []*LocationResponse `json:"locations"`
}

// ErrorResponse is the standard error response envelope
type ErrorResponse struct {
	Error string `json:"error"`
}

// ToLocationResponse maps a domain entity to a presenter DTO
func ToLocationResponse(loc *vehicle.Location) *LocationResponse {
	return &LocationResponse{
		VehicleID: loc.VehicleID,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		Timestamp: loc.Timestamp,
	}
}

// ToHistoryResponse maps a slice of domain entities to a presenter DTO
func ToHistoryResponse(vehicleID string, locs []*vehicle.Location) *HistoryResponse {
	responses := make([]*LocationResponse, 0, len(locs))
	for _, loc := range locs {
		responses = append(responses, ToLocationResponse(loc))
	}
	return &HistoryResponse{
		VehicleID: vehicleID,
		Total:     len(responses),
		Locations: responses,
	}
}
