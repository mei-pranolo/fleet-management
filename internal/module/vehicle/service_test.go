package vehicle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── test DB setup ─────────────────────────────────────────────────────────────

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&VehicleLocationModel{}))
	return db
}

func seedLocation(t *testing.T, repo Repository, loc *Location) {
	t.Helper()
	require.NoError(t, repo.Save(loc))
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestRepository_Save(t *testing.T) {
	tests := []struct {
		name    string
		loc     *Location
		wantErr bool
	}{
		{
			name: "success",
			loc:  validLocation(),
		},
		{
			name: "save multiple times same vehicle",
			loc:  &Location{VehicleID: "B1234XYZ", Latitude: -6.21, Longitude: 106.85, Timestamp: 1715000001},
		},
		{
			name: "different vehicle",
			loc:  &Location{VehicleID: "B9999DEF", Latitude: -6.23, Longitude: 106.84, Timestamp: 1715000002},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewRepository(newTestDB(t))
			err := repo.Save(tt.loc)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ── FindLatest ────────────────────────────────────────────────────────────────

func TestRepository_FindLatest(t *testing.T) {
	tests := []struct {
		name          string
		seed          []*Location
		queryVehicle  string
		wantTimestamp int64
		wantErr       bool
	}{
		{
			name: "returns most recent record",
			seed: []*Location{
				{VehicleID: "B1234XYZ", Latitude: -6.20, Longitude: 106.84, Timestamp: 1715000000},
				{VehicleID: "B1234XYZ", Latitude: -6.21, Longitude: 106.85, Timestamp: 1715003456},
			},
			queryVehicle:  "B1234XYZ",
			wantTimestamp: 1715003456,
		},
		{
			name:         "vehicle not found",
			seed:         []*Location{},
			queryVehicle: "UNKNOWN",
			wantErr:      true,
		},
		{
			name: "isolated by vehicle_id",
			seed: []*Location{
				{VehicleID: "AAA", Latitude: -6.10, Longitude: 106.70, Timestamp: 1715000001},
				{VehicleID: "BBB", Latitude: -6.20, Longitude: 106.80, Timestamp: 1715000099},
			},
			queryVehicle:  "AAA",
			wantTimestamp: 1715000001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewRepository(newTestDB(t))
			for _, loc := range tt.seed {
				seedLocation(t, repo, loc)
			}

			got, err := repo.FindLatest(tt.queryVehicle)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTimestamp, got.Timestamp)
				assert.Equal(t, tt.queryVehicle, got.VehicleID)
			}
		})
	}
}

// ── FindByTimeRange ───────────────────────────────────────────────────────────

func TestRepository_FindByTimeRange(t *testing.T) {
	tests := []struct {
		name         string
		seed         []*Location
		queryVehicle string
		start        int64
		end          int64
		wantLen      int
		wantOrdered  bool
	}{
		{
			name: "returns records within range",
			seed: []*Location{
				{VehicleID: "B1234XYZ", Latitude: -6.20, Longitude: 106.84, Timestamp: 1715000010},
				{VehicleID: "B1234XYZ", Latitude: -6.21, Longitude: 106.85, Timestamp: 1715000020},
				{VehicleID: "B1234XYZ", Latitude: -6.22, Longitude: 106.86, Timestamp: 1715000030},
			},
			queryVehicle: "B1234XYZ",
			start:        1715000010,
			end:          1715000020,
			wantLen:      2,
		},
		{
			name: "inclusive bounds — start and end included",
			seed: []*Location{
				{VehicleID: "B1234XYZ", Latitude: -6.20, Longitude: 106.84, Timestamp: 1715000000},
				{VehicleID: "B1234XYZ", Latitude: -6.21, Longitude: 106.85, Timestamp: 1715009999},
			},
			queryVehicle: "B1234XYZ",
			start:        1715000000,
			end:          1715009999,
			wantLen:      2,
		},
		{
			name: "no results outside range",
			seed: []*Location{
				{VehicleID: "B1234XYZ", Latitude: -6.20, Longitude: 106.84, Timestamp: 1715000010},
			},
			queryVehicle: "B1234XYZ",
			start:        1716000000,
			end:          1716009999,
			wantLen:      0,
		},
		{
			name: "ordered ascending by timestamp",
			seed: []*Location{
				{VehicleID: "B1234XYZ", Latitude: -6.22, Longitude: 106.86, Timestamp: 1715000030},
				{VehicleID: "B1234XYZ", Latitude: -6.20, Longitude: 106.84, Timestamp: 1715000010},
			},
			queryVehicle: "B1234XYZ",
			start:        1715000000,
			end:          1715000099,
			wantLen:      2,
			wantOrdered:  true,
		},
		{
			name: "isolated by vehicle_id",
			seed: []*Location{
				{VehicleID: "AAA", Latitude: -6.20, Longitude: 106.84, Timestamp: 1715000010},
				{VehicleID: "BBB", Latitude: -6.21, Longitude: 106.85, Timestamp: 1715000020},
			},
			queryVehicle: "AAA",
			start:        1715000000,
			end:          1715000099,
			wantLen:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewRepository(newTestDB(t))
			for _, loc := range tt.seed {
				seedLocation(t, repo, loc)
			}

			got, err := repo.FindByTimeRange(tt.queryVehicle, tt.start, tt.end)

			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)

			if tt.wantOrdered && len(got) > 1 {
				for i := 1; i < len(got); i++ {
					assert.Less(t, got[i-1].Timestamp, got[i].Timestamp,
						"expected ASC order at index %d", i)
				}
			}
		})
	}
}