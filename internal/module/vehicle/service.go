package vehicle

import (
	"fmt"
	"math"

	"fleet-management/internal/config"
)

// GeofencePublisher defines the contract for publishing geofence events
type GeofencePublisher interface {
	Publish(event *GeofenceEvent) error
}

// Service handles all business logic related to vehicle locations
type Service struct {
	repo      Repository
	publisher GeofencePublisher
	geofences []config.GeofencePoint
}

// NewService creates a new vehicle location Service
func NewService(repo Repository, publisher GeofencePublisher, geofences []config.GeofencePoint) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
		geofences: geofences,
	}
}

// ProcessLocation validates, persists, and runs geofence checks for an incoming location
func (s *Service) ProcessLocation(loc *Location) error {
	if err := validate(loc); err != nil {
		return fmt.Errorf("invalid location: %w", err)
	}
	if err := s.repo.Save(loc); err != nil {
		return fmt.Errorf("failed to save location: %w", err)
	}
	s.checkGeofences(loc)
	return nil
}

// GetLatestLocation returns the most recent location for a given vehicle
func (s *Service) GetLatestLocation(vehicleID string) (*Location, error) {
	if vehicleID == "" {
		return nil, fmt.Errorf("vehicle_id is required")
	}
	return s.repo.FindLatest(vehicleID)
}

// GetLocationHistory returns all location records within the given unix timestamp range
func (s *Service) GetLocationHistory(vehicleID string, start, end int64) ([]*Location, error) {
	if vehicleID == "" {
		return nil, fmt.Errorf("vehicle_id is required")
	}
	if start > end {
		return nil, fmt.Errorf("start must be before end")
	}
	return s.repo.FindByTimeRange(vehicleID, start, end)
}

// checkGeofences evaluates all configured geofence points against the vehicle location
func (s *Service) checkGeofences(loc *Location) {
	for _, gf := range s.geofences {
		if haversine(loc.Latitude, loc.Longitude, gf.Latitude, gf.Longitude) <= gf.RadiusMeters {
			_ = s.publisher.Publish(&GeofenceEvent{
				VehicleID: loc.VehicleID,
				Event:     "geofence_entry",
				Location:  LocationPayload{Latitude: loc.Latitude, Longitude: loc.Longitude},
				Timestamp: loc.Timestamp,
			})
		}
	}
}

// validate checks required fields and coordinate bounds
func validate(loc *Location) error {
	if loc.VehicleID == "" {
		return fmt.Errorf("vehicle_id is required")
	}
	if loc.Latitude < -90 || loc.Latitude > 90 {
		return fmt.Errorf("latitude out of range")
	}
	if loc.Longitude < -180 || loc.Longitude > 180 {
		return fmt.Errorf("longitude out of range")
	}
	if loc.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	return nil
}

const earthRadius = 6371000.0

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func toRad(deg float64) float64 { return deg * math.Pi / 180 }
