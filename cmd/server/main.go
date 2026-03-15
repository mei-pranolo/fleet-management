package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"fleet-management/internal/adapter/http/handler"
	"fleet-management/internal/adapter/http/route"
	mqttadapter "fleet-management/internal/adapter/mqtt"
	rmqadapter "fleet-management/internal/adapter/rabbitmq"
	"fleet-management/internal/config"
	"fleet-management/internal/infrastructure/database"
	"fleet-management/internal/infrastructure/messagebroker"
	"fleet-management/internal/module/vehicle"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatalf("[Main] Failed to load config: %v", err)
	}

	// ── Database (GORM) ───────────────────────────────────────────────────────
	db, err := database.NewGormDB(&cfg.Database)
	if err != nil {
		log.Fatalf("[Main] Failed to connect to PostgreSQL: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := database.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("[Main] Failed to run migrations: %v", err)
	}
	log.Println("[Main] Migrations applied successfully")

	// ── RabbitMQ ──────────────────────────────────────────────────────────────
	rmqClient, err := messagebroker.NewRabbitMQClient(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("[Main] Failed to connect to RabbitMQ: %v", err)
	}
	defer rmqClient.Close()

	// ── Vehicle module wiring ─────────────────────────────────────────────────
	vehicleRepo := vehicle.NewRepository(db)
	vehicleSvc := vehicle.NewService(vehicleRepo, rmqClient, cfg.Geofence.Points)

	// ── MQTT subscriber ───────────────────────────────────────────────────────
	mqttSub, err := mqttadapter.NewSubscriber(&cfg.MQTT, vehicleSvc)
	if err != nil {
		log.Fatalf("[Main] Failed to connect to MQTT broker: %v", err)
	}
	defer mqttSub.Disconnect()

	if err := mqttSub.Subscribe(); err != nil {
		log.Fatalf("[Main] Failed to subscribe to MQTT topic: %v", err)
	}

	// ── RabbitMQ geofence worker ──────────────────────────────────────────────
	worker := rmqadapter.NewWorker(rmqClient)
	if err := worker.Start(); err != nil {
		log.Fatalf("[Main] Failed to start RabbitMQ worker: %v", err)
	}

	// ── HTTP server (Fiber) ───────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		AppName: cfg.App.Name,
	})

	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(logger.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": cfg.App.Name})
	})

	vehicleHandler := handler.NewVehicleHandler(vehicleSvc)
	route.Register(app, vehicleHandler)

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("[Main] Shutdown signal received...")
		if err := app.Shutdown(); err != nil {
			log.Printf("[Main] Error during shutdown: %v", err)
		}
	}()

	addr := ":" + cfg.App.Port
	log.Printf("[Main] %s listening on %s", cfg.App.Name, addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("[Main] Server error: %v", err)
	}
}
