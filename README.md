# Fleet Management Backend

Backend service for a vehicle fleet management system built with **Go**, **Fiber**, **GORM**, **MQTT**, **PostgreSQL**, and **RabbitMQ** — fully containerised with Docker Compose.

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        Docker Compose                            │
│                                                                  │
│  ┌─────────────┐     MQTT      ┌──────────────────────────────┐  │
│  │fleet-feeder │──────────────▶│       fleet-service          │  │
│  │  (Go script)│               │                              │  │
│  └─────────────┘               │  cmd/server/main.go          │  │
│                                │  ┌──────────────────────┐    │  │
│  ┌─────────────┐               │  │ adapter/mqtt         │    │  │
│  │  mosquitto  │◀──────────────│  │  subscriber.go       │    │  │
│  │ (MQTT broker│               │  ├──────────────────────┤    │  │
│  └─────────────┘               │  │ adapter/http         │    │  │
│                                │  │  handler / route /   │    │  │
│  ┌─────────────┐               │  │  presenter           │    │  │
│  │  postgres   │◀──────────────│  ├──────────────────────┤    │  │
│  └─────────────┘               │  │ module/vehicle       │    │  │
│                                │  │  entity   model      │    │  │
│  ┌─────────────┐               │  │  service  repository │    │  │
│  │  rabbitmq   │◀──────────────│  ├──────────────────────┤    │  │
│  └─────────────┘               │  │ infrastructure/      │    │  │
│                                │  │  database / broker   │    │  │
│                                │  └──────────────────────┘    │  │
│                                └──────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### Layer breakdown

| Layer | Package | Responsibility |
|---|---|---|
| Adapter – HTTP | `internal/adapter/http/{handler,route,presenter}` | Receive HTTP requests, format responses |
| Adapter – MQTT | `internal/adapter/mqtt` | Subscribe to broker, parse payloads |
| Adapter – RabbitMQ | `internal/adapter/rabbitmq` | Geofence event worker |
| Module – Vehicle | `internal/module/vehicle` | Entity, GORM model, Repository interface + impl, Service |
| Infrastructure – DB | `internal/infrastructure/database` | GORM connection, SQL migration loader |
| Infrastructure – MQ | `internal/infrastructure/messagebroker` | RabbitMQ client |
| Config | `internal/config` | Viper loader, typed config structs |

---

## Project Structure

```
fleet-management/
├── cmd/
│   ├── server/
│   │   └── main.go                        # Service entrypoint
│   └── feeder/
│       ├── main.go                        # Mock data publisher
│       └── feeder_config.yml
│
├── internal/
│   ├── config/
│   │   └── config.go                      # Viper config loader
│   │
│   ├── module/
│   │   └── vehicle/                       # One package per module
│   │       ├── entity.go                  # Domain entity + conversion helpers
│   │       ├── model.go                   # GORM model (VehicleLocationModel)
│   │       ├── repository.go              # Repository interface + GORM impl
│   │       └── service.go                 # Business logic, geofence, validation
│   │
│   ├── adapter/
│   │   ├── http/
│   │   │   ├── handler/
│   │   │   │   └── vehicle_handler.go
│   │   │   ├── route/
│   │   │   │   └── route.go
│   │   │   └── presenter/
│   │   │       └── vehicle_presenter.go
│   │   ├── mqtt/
│   │   │   └── subscriber.go
│   │   └── rabbitmq/
│   │       └── worker.go
│   │
│   └── infrastructure/
│       ├── database/
│       │   └── postgres.go                # GORM connection + SQL migration loader
│       └── messagebroker/
│           └── rabbitmq.go
│
├── migrations/
│   └── 0001_create_vehicle_locations.sql  # Add new files here for future migrations
│
├── deployments/
│   └── mosquitto/
│       └── config/
│           └── mosquitto.conf
│
├── config.yml
├── docker-compose.yml
├── Dockerfile
├── Dockerfile.feeder
├── go.mod
└── go.sum
```

---

## Module Structure (`internal/module/vehicle`)

All files share `package vehicle`, so unexported helpers are accessible across all files in the module.

| File | Contents |
|---|---|
| `entity.go` | `Location`, `GeofenceEvent`, `LocationPayload` + `toModel()` / `fromModel()` converters |
| `model.go` | `VehicleLocationModel` — GORM struct with column tags and `TableName()` |
| `repository.go` | `Repository` interface + `gormRepository` implementation |
| `service.go` | `Service`, `GeofencePublisher` interface, haversine distance, input validation |

To add a new module (e.g. `driver`), create `internal/module/driver/` with the same pattern. Modules can depend on each other by injecting the other module's `*Service`.

---

## Migrations

Migration files live in `migrations/` and are applied in **lexicographic order** at startup via `database.RunMigrations()`. The loader reads every `*.sql` file and executes it. All DDL statements use `IF NOT EXISTS` guards so re-running is safe.

```
migrations/
├── 0001_create_vehicle_locations.sql
├── 0002_add_speed_column.sql          ← example future migration
└── 0003_create_drivers_table.sql      ← another module
```

To add a migration, create a new numbered `.sql` file — no code changes required.

---

## Configuration (`config.yml`)

All connection settings are centralised in `config.yml` and loaded via **Viper**. Any value can be overridden with an environment variable using `_` as the separator:

```bash
DATABASE_HOST=my-host DATABASE_PORT=5433 ./server
```

Key sections:

```yaml
app:
  name: "fleet-management"
  port: "8080"

database:   # PostgreSQL host/port/credentials + pool settings
mqtt:       # broker URL, client ID, topic prefix, QoS
rabbitmq:   # AMQP URL, exchange, queue, routing key
geofence:   # list of { name, latitude, longitude, radius_meters }
```

---

## Running with Docker Compose

```bash
# Build and start all containers
docker compose up --build

# Detached mode
docker compose up --build -d

# View logs
docker compose logs -f fleet-service
docker compose logs -f fleet-feeder

# Tear down
docker compose down

# Tear down and remove volumes
docker compose down -v
```

### Start-up order

```
mosquitto ──healthy──▶ fleet-service ──started──▶ fleet-feeder
postgres  ──healthy──▶ fleet-service
rabbitmq  ──healthy──▶ fleet-service
```

---

## API Reference

### `GET /health`
```json
{ "status": "ok", "service": "fleet-management" }
```

### `GET /vehicles/:vehicle_id/location`

Returns the most recent location for a vehicle.

```bash
curl http://localhost:8080/vehicles/B1234XYZ/location
```

**200 OK**
```json
{
  "vehicle_id": "B1234XYZ",
  "latitude": -6.208812,
  "longitude": 106.845623,
  "timestamp": 1715003456
}
```

### `GET /vehicles/:vehicle_id/history?start=<unix>&end=<unix>`

Returns all location records within the given unix timestamp range.

```bash
curl "http://localhost:8080/vehicles/B1234XYZ/history?start=1715000000&end=1715009999"
```

**200 OK**
```json
{
  "vehicle_id": "B1234XYZ",
  "total": 2,
  "locations": [
    { "vehicle_id": "B1234XYZ", "latitude": -6.2088, "longitude": 106.8456, "timestamp": 1715000010 },
    { "vehicle_id": "B1234XYZ", "latitude": -6.2089, "longitude": 106.8457, "timestamp": 1715000012 }
  ]
}
```

---

## MQTT

- **Broker**: `mosquitto:1883`
- **Topic pattern**: `/fleet/vehicle/{vehicle_id}/location`
- **QoS**: 1

**Payload**
```json
{
  "vehicle_id": "B1234XYZ",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "timestamp": 1715003456
}
```

Manual publish for testing:
```bash
mosquitto_pub -h localhost -p 1883 \
  -t "/fleet/vehicle/B1234XYZ/location" \
  -m '{"vehicle_id":"B1234XYZ","latitude":-6.2088,"longitude":106.8456,"timestamp":1715003456}'
```

---

## Geofence & RabbitMQ

When a vehicle comes within **50 metres** of a configured geofence point, an event is published:

- **Exchange**: `fleet.events`
- **Queue**: `geofence_alerts`
- **Routing key**: `geofence`

```json
{
  "vehicle_id": "B1234XYZ",
  "event": "geofence_entry",
  "location": { "latitude": -6.2088, "longitude": 106.8456 },
  "timestamp": 1715003456
}
```

RabbitMQ Management UI → [http://localhost:15672](http://localhost:15672) (guest / guest)

Geofence points are configured in `config.yml`:
```yaml
geofence:
  points:
    - name: "Monas"
      latitude: -6.1754
      longitude: 106.8272
      radius_meters: 50
```

---

## Database

Table is created automatically on first startup by the migration loader.

```sql
CREATE TABLE vehicle_locations (
    id          BIGSERIAL        PRIMARY KEY,
    vehicle_id  VARCHAR(50)      NOT NULL,
    latitude    DOUBLE PRECISION NOT NULL,
    longitude   DOUBLE PRECISION NOT NULL,
    timestamp   BIGINT           NOT NULL,
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);
```

Connect directly:
```bash
docker exec -it fleet-postgres psql -U fleet_user -d fleet_db
```
