package vehicle

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VehicleLocationModel is the GORM model that maps to the vehicle_locations table
type VehicleLocationModel struct {
	ID        string    `gorm:"column:id;type:uuid;primaryKey"`
	VehicleID string    `gorm:"column:vehicle_id;type:varchar(50);not null"`
	Latitude  float64   `gorm:"column:latitude;not null"`
	Longitude float64   `gorm:"column:longitude;not null"`
	Timestamp int64     `gorm:"column:timestamp;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (VehicleLocationModel) TableName() string {
	return "vehicle_locations"
}

// BeforeCreate generates a UUID before inserting — works on any DB driver
func (m *VehicleLocationModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}