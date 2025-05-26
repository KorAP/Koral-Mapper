package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/KorAP/KoralPipe-TermMapper/config"
	"github.com/KorAP/KoralPipe-TermMapper/mapper"
	"github.com/alecthomas/kong"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	maxInputLength = 1024 * 1024 // 1MB
	maxParamLength = 1024        // 1KB
)

type appConfig struct {
	Port     int    `kong:"short='p',default='8080',help='Port to listen on'"`
	Config   string `kong:"short='c',required,help='YAML configuration file containing mapping directives'"`
	LogLevel string `kong:"short='l',default='info',help='Log level (debug, info, warn, error)'"`
}

func parseConfig() *appConfig {
	cfg := &appConfig{}
	ctx := kong.Parse(cfg,
		kong.Description("A web service for transforming JSON objects using term mapping rules."),
		kong.UsageOnError(),
	)
	if ctx.Error != nil {
		fmt.Fprintln(os.Stderr, ctx.Error)
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
	cfg := parseConfig()

	// Set up logging
	setupLogger(cfg.LogLevel)

	// Load configuration file
	yamlConfig, err := config.LoadConfig(cfg.Config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Create a new mapper instance
	m, err := mapper.NewMapper(yamlConfig.Lists)
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
		log.Info().Int("port", cfg.Port).Msg("Starting server")
		if err := app.Listen(fmt.Sprintf(":%d", cfg.Port)); err != nil {
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
	// Define parameter checks
	params := []struct {
		name  string
		value string
	}{
		{"mapID", mapID},
		{"dir", dir},
		{"foundryA", foundryA},
		{"foundryB", foundryB},
		{"layerA", layerA},
		{"layerB", layerB},
	}

	for _, param := range params {
		// Check input lengths
		if len(param.value) > maxParamLength {
			return fmt.Errorf("%s too long (max %d bytes)", param.name, maxParamLength)
		}
		// Check for invalid characters in parameters
		if strings.ContainsAny(param.value, "<>{}[]\\") {
			return fmt.Errorf("%s contains invalid characters", param.name)
		}
	}

	if len(body) > maxInputLength {
		return fmt.Errorf("request body too large (max %d bytes)", maxInputLength)
	}

	return nil
}
