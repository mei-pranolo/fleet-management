package vehicle

import (
	"fmt"

	"gorm.io/gorm"
)

// Repository defines the persistence contract for vehicle locations
type Repository interface {
	Save(loc *Location) error
	FindLatest(vehicleID string) (*Location, error)
	FindByTimeRange(vehicleID string, start, end int64) ([]*Location, error)
}

// gormRepository is the GORM-backed implementation of Repository
type gormRepository struct {
	db *gorm.DB
}

// NewRepository creates a new GORM-backed vehicle location repository
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Save(loc *Location) error {
	model := loc.toModel()
	if err := r.db.Create(model).Error; err != nil {
		return fmt.Errorf("failed to save vehicle location: %w", err)
	}
	return nil
}

func (r *gormRepository) FindLatest(vehicleID string) (*Location, error) {
	var model VehicleLocationModel
	err := r.db.
		Where("vehicle_id = ?", vehicleID).
		Order("timestamp DESC").
		First(&model).Error

	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("no location found for vehicle %s", vehicleID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query latest location: %w", err)
	}

	return fromModel(&model), nil
}

func (r *gormRepository) FindByTimeRange(vehicleID string, start, end int64) ([]*Location, error) {
	var models []VehicleLocationModel
	err := r.db.
		Where("vehicle_id = ? AND timestamp BETWEEN ? AND ?", vehicleID, start, end).
		Order("timestamp ASC").
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query location history: %w", err)
	}

	locations := make([]*Location, 0, len(models))
	for i := range models {
		locations = append(locations, fromModel(&models[i]))
	}
	return locations, nil
}
