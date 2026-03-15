package vehicle

import (
	"errors"
	"testing"

	"fleet-management/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func validLocation() *Location {
	return &Location{
		VehicleID: "B1234XYZ",
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Timestamp: 1715003456,
	}
}

func defaultGeofences() []config.GeofencePoint {
	return []config.GeofencePoint{
		{Name: "TestZone", Latitude: -6.2088, Longitude: 106.8456, RadiusMeters: 50},
	}
}

// ── ProcessLocation ───────────────────────────────────────────────────────────

func TestProcessLocation(t *testing.T) {
	tests := []struct {
		name          string
		loc           *Location
		saveErr       error
		geofences     []config.GeofencePoint
		wantErr       string
		wantPublished int
	}{
		{
			name:          "success with geofence triggered",
			loc:           validLocation(),
			geofences:     defaultGeofences(),
			wantPublished: 1,
		},
		{
			name:    "empty vehicle_id",
			loc:     &Location{Latitude: -6.2088, Longitude: 106.8456, Timestamp: 1715003456},
			wantErr: "vehicle_id is required",
		},
		{
			name:    "latitude out of range",
			loc:     &Location{VehicleID: "B1234XYZ", Latitude: 999, Longitude: 106.8456, Timestamp: 1715003456},
			wantErr: "latitude out of range",
		},
		{
			name:    "longitude out of range",
			loc:     &Location{VehicleID: "B1234XYZ", Latitude: -6.2088, Longitude: -999, Timestamp: 1715003456},
			wantErr: "longitude out of range",
		},
		{
			name:    "zero timestamp",
			loc:     &Location{VehicleID: "B1234XYZ", Latitude: -6.2088, Longitude: 106.8456, Timestamp: 0},
			wantErr: "timestamp must be positive",
		},
		{
			name:    "repo save error",
			loc:     validLocation(),
			saveErr: errors.New("db down"),
			wantErr: "failed to save location",
		},
		{
			name:          "geofence not triggered — vehicle too far",
			loc:           validLocation(),
			geofences:     []config.GeofencePoint{{Name: "FarZone", Latitude: -6.1754, Longitude: 106.8272, RadiusMeters: 50}},
			wantPublished: 0,
		},
		{
			name: "geofence boundary edge — ~50m north",
			loc: &Location{
				VehicleID: "B1234XYZ",
				Latitude:  -6.2088 + 0.000449,
				Longitude: 106.8456,
				Timestamp: 1715003456,
			},
			geofences:     defaultGeofences(),
			wantPublished: 1,
		},
		{
			name: "multiple geofences both hit",
			loc:  validLocation(),
			geofences: []config.GeofencePoint{
				{Name: "Zone1", Latitude: -6.2088, Longitude: 106.8456, RadiusMeters: 50},
				{Name: "Zone2", Latitude: -6.2088, Longitude: 106.8456, RadiusMeters: 100},
			},
			wantPublished: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockRepository(t)
			pub := NewMockGeofencePublisher(t)

			if tt.wantErr == "" || tt.saveErr != nil {
				// Save is only called when validation passes
				if tt.wantErr == "" || tt.saveErr != nil {
					repo.On("Save", tt.loc).Return(tt.saveErr)
				}
			}

			for i := 0; i < tt.wantPublished; i++ {
				pub.On("Publish", &GeofenceEvent{
					VehicleID: tt.loc.VehicleID,
					Event:     "geofence_entry",
					Location:  LocationPayload{Latitude: tt.loc.Latitude, Longitude: tt.loc.Longitude},
					Timestamp: tt.loc.Timestamp,
				}).Return(nil).Once()
			}

			svc := NewService(repo, pub, tt.geofences)
			err := svc.ProcessLocation(tt.loc)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ── GetLatestLocation ─────────────────────────────────────────────────────────

func TestGetLatestLocation(t *testing.T) {
	tests := []struct {
		name       string
		vehicleID  string
		repoResult *Location
		repoErr    error
		wantErr    string
	}{
		{
			name:       "success",
			vehicleID:  "B1234XYZ",
			repoResult: validLocation(),
		},
		{
			name:      "empty vehicle_id",
			vehicleID: "",
			wantErr:   "vehicle_id is required",
		},
		{
			name:      "repo returns error",
			vehicleID: "B1234XYZ",
			repoErr:   errors.New("not found"),
			wantErr:   "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockRepository(t)
			pub := NewMockGeofencePublisher(t)

			if tt.vehicleID != "" {
				repo.On("FindLatest", tt.vehicleID).Return(tt.repoResult, tt.repoErr)
			}

			svc := NewService(repo, pub, nil)
			got, err := svc.GetLatestLocation(tt.vehicleID)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.repoResult.VehicleID, got.VehicleID)
			}
		})
	}
}

// ── GetLocationHistory ────────────────────────────────────────────────────────

func TestGetLocationHistory(t *testing.T) {
	tests := []struct {
		name       string
		vehicleID  string
		start      int64
		end        int64
		repoResult []*Location
		repoErr    error
		wantErr    string
		wantLen    int
	}{
		{
			name:       "success with results",
			vehicleID:  "B1234XYZ",
			start:      1715000000,
			end:        1715009999,
			repoResult: []*Location{validLocation(), validLocation()},
			wantLen:    2,
		},
		{
			name:       "success empty result",
			vehicleID:  "B1234XYZ",
			start:      1715000000,
			end:        1715009999,
			repoResult: []*Location{},
			wantLen:    0,
		},
		{
			name:      "empty vehicle_id",
			vehicleID: "",
			start:     1715000000,
			end:       1715009999,
			wantErr:   "vehicle_id is required",
		},
		{
			name:      "start after end",
			vehicleID: "B1234XYZ",
			start:     1715009999,
			end:       1715000000,
			wantErr:   "start must be before end",
		},
		{
			name:      "repo returns error",
			vehicleID: "B1234XYZ",
			start:     1715000000,
			end:       1715009999,
			repoErr:   errors.New("db error"),
			wantErr:   "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockRepository(t)
			pub := NewMockGeofencePublisher(t)

			if tt.vehicleID != "" && tt.start <= tt.end {
				repo.On("FindByTimeRange", tt.vehicleID, tt.start, tt.end).
					Return(tt.repoResult, tt.repoErr)
			}

			svc := NewService(repo, pub, nil)
			got, err := svc.GetLocationHistory(tt.vehicleID, tt.start, tt.end)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
			}
		})
	}
}

// ── Haversine ─────────────────────────────────────────────────────────────────

func TestHaversine(t *testing.T) {
	tests := []struct {
		name        string
		lat1, lon1  float64
		lat2, lon2  float64
		wantMeters  float64
		deltaMeters float64
	}{
		{
			name: "same point",
			lat1: -6.2088, lon1: 106.8456,
			lat2: -6.2088, lon2: 106.8456,
			wantMeters:  0,
			deltaMeters: 0,
		},
		{
			name: "Monas to Sudirman ~4km",
			lat1: -6.1754, lon1: 106.8272,
			lat2: -6.2088, lon2: 106.8182,
			wantMeters:  4000,
			deltaMeters: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := haversine(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			assert.InDelta(t, tt.wantMeters, dist, tt.deltaMeters)
		})
	}
}
