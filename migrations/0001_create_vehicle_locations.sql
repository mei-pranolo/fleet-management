-- Migration: 0001_create_vehicle_locations
-- Creates the vehicle_locations table with UUID primary key

CREATE TABLE IF NOT EXISTS vehicle_locations (
    id          UUID             PRIMARY KEY,
    vehicle_id  VARCHAR(50)      NOT NULL,
    latitude    DOUBLE PRECISION NOT NULL,
    longitude   DOUBLE PRECISION NOT NULL,
    timestamp   BIGINT           NOT NULL,
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);