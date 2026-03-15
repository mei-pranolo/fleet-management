package vehicle

// Location is the core domain entity for a vehicle GPS reading
type Location struct {
	VehicleID string  `json:"vehicle_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

// GeofenceEvent represents an event triggered when a vehicle enters a geofence area
type GeofenceEvent struct {
	VehicleID string          `json:"vehicle_id"`
	Event     string          `json:"event"`
	Location  LocationPayload `json:"location"`
	Timestamp int64           `json:"timestamp"`
}

// LocationPayload is the nested coordinate inside a GeofenceEvent
type LocationPayload struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// toModel converts a domain entity to a GORM model.
// ID is left empty — PostgreSQL generates it via gen_random_uuid().
func (l *Location) toModel() *VehicleLocationModel {
	return &VehicleLocationModel{
		VehicleID: l.VehicleID,
		Latitude:  l.Latitude,
		Longitude: l.Longitude,
		Timestamp: l.Timestamp,
	}
}

// fromModel converts a GORM model back to a domain entity
func fromModel(m *VehicleLocationModel) *Location {
	return &Location{
		VehicleID: m.VehicleID,
		Latitude:  m.Latitude,
		Longitude: m.Longitude,
		Timestamp: m.Timestamp,
	}
}
