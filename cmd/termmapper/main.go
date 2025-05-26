package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/KorAP/KoralPipe-TermMapper/mapper"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	maxInputLength = 1024 * 1024 // 1MB
	maxParamLength = 1024        // 1KB
)

type config struct {
	port     int
	config   string
	logLevel string
}

func parseFlags() *config {
	cfg := &config{}

	flag.IntVar(&cfg.port, "port", 8080, "Port to listen on")
	flag.IntVar(&cfg.port, "p", 8080, "Port to listen on (shorthand)")

	flag.StringVar(&cfg.config, "config", "", "YAML configuration file containing mapping directives")
	flag.StringVar(&cfg.config, "c", "", "YAML configuration file containing mapping directives (shorthand)")

	flag.StringVar(&cfg.logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.StringVar(&cfg.logLevel, "l", "info", "Log level (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nA web service for transforming JSON objects using term mapping rules.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if cfg.config == "" {
		fmt.Fprintln(os.Stderr, "Error: config file is required")
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

func setupLogger(level string) {
	// Parse log level
	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		log.Error().Err(err).Str("level", level).Msg("Invalid log level, defaulting to info")
		lvl = zerolog.InfoLevel
	}

	// Configure zerolog
	zerolog.SetGlobalLevel(lvl)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	// Parse command line flags
	cfg := parseFlags()

	// Set up logging
	setupLogger(cfg.logLevel)

	// Create a new mapper instance
	m, err := mapper.NewMapper(cfg.config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create mapper")
	}

	// Create fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		BodyLimit:             maxInputLength,
	})

	// Set up routes
	setupRoutes(app, m)

	// Start server
	go func() {
		log.Info().Int("port", cfg.port).Msg("Starting server")
		if err := app.Listen(fmt.Sprintf(":%d", cfg.port)); err != nil {
			log.Fatal().Err(err).Msg("Server error")
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Info().Msg("Shutting down server")
	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Error during shutdown")
	}
}

func setupRoutes(app *fiber.App, m *mapper.Mapper) {
	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Transformation endpoint
	app.Post("/:map/query", handleTransform(m))
}

func handleTransform(m *mapper.Mapper) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get parameters
		mapID := c.Params("map")
		dir := c.Query("dir", "atob")
		foundryA := c.Query("foundryA", "")
		foundryB := c.Query("foundryB", "")
		layerA := c.Query("layerA", "")
		layerB := c.Query("layerB", "")

		// Validate input parameters
		if err := validateInput(mapID, dir, foundryA, foundryB, layerA, layerB, c.Body()); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Validate direction
		if dir != "atob" && dir != "btoa" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid direction, must be 'atob' or 'btoa'",
			})
		}

		// Parse request body
		var jsonData any
		if err := c.BodyParser(&jsonData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON in request body",
			})
		}

		// Apply mappings
		result, err := m.ApplyMappings(mapID, mapper.MappingOptions{
			Direction: mapper.Direction(dir),
			FoundryA:  foundryA,
			FoundryB:  foundryB,
			LayerA:    layerA,
			LayerB:    layerB,
		}, jsonData)

		if err != nil {
			log.Error().Err(err).
				Str("mapID", mapID).
				Str("direction", dir).
				Msg("Failed to apply mappings")

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(result)
	}
}

// validateInput checks if the input parameters are valid
func validateInput(mapID, dir, foundryA, foundryB, layerA, layerB string, body []byte) error {
	// Check input lengths
	if len(mapID) > maxParamLength {
		return fmt.Errorf("map ID too long (max %d bytes)", maxParamLength)
	}
	if len(dir) > maxParamLength {
		return fmt.Errorf("direction too long (max %d bytes)", maxParamLength)
	}
	if len(foundryA) > maxParamLength {
		return fmt.Errorf("foundryA too long (max %d bytes)", maxParamLength)
	}
	if len(foundryB) > maxParamLength {
		return fmt.Errorf("foundryB too long (max %d bytes)", maxParamLength)
	}
	if len(layerA) > maxParamLength {
		return fmt.Errorf("layerA too long (max %d bytes)", maxParamLength)
	}
	if len(layerB) > maxParamLength {
		return fmt.Errorf("layerB too long (max %d bytes)", maxParamLength)
	}
	if len(body) > maxInputLength {
		return fmt.Errorf("request body too large (max %d bytes)", maxInputLength)
	}

	// Check for invalid characters in parameters
	for _, param := range []struct {
		name  string
		value string
	}{
		{"mapID", mapID},
		{"dir", dir},
		{"foundryA", foundryA},
		{"foundryB", foundryB},
		{"layerA", layerA},
		{"layerB", layerB},
	} {
		if strings.ContainsAny(param.value, "<>{}[]\\") {
			return fmt.Errorf("%s contains invalid characters", param.name)
		}
	}

	return nil
}
