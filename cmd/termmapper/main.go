package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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
	Port     *int     `kong:"short='p',help='Port to listen on'"`
	Config   string   `kong:"short='c',help='YAML configuration file containing mapping directives and global settings'"`
	Mappings []string `kong:"short='m',help='Individual YAML mapping files to load (supports glob patterns like dir/*.yaml)'"`
	LogLevel *string  `kong:"short='l',help='Log level (debug, info, warn, error)'"`
}

type TemplateMapping struct {
	ID          string
	Description string
}

// TemplateData holds data for the Kalamar plugin template
type TemplateData struct {
	Title       string
	Version     string
	Hash        string
	Date        string
	Description string
	Server      string
	SDK         string
	MapID       string
	Mappings    []TemplateMapping
}

func parseConfig() *appConfig {
	cfg := &appConfig{}

	desc := config.Description
	desc += " [" + config.Version + "]"

	ctx := kong.Parse(cfg,
		kong.Description(desc),
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

	// Validate command line arguments
	if cfg.Config == "" && len(cfg.Mappings) == 0 {
		log.Fatal().Msg("At least one configuration source must be provided: use -c for main config file or -m for mapping files")
	}

	// Expand glob patterns in mapping files
	expandedMappings, err := expandGlobs(cfg.Mappings)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to expand glob patterns in mapping files")
	}

	// Load configuration from multiple sources
	yamlConfig, err := config.LoadFromSources(cfg.Config, expandedMappings)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	finalPort := yamlConfig.Port
	finalLogLevel := yamlConfig.LogLevel

	// Use command line values if provided (they override config file)
	if cfg.Port != nil {
		finalPort = *cfg.Port
	}
	if cfg.LogLevel != nil {
		finalLogLevel = *cfg.LogLevel
	}

	// Set up logging with the final log level
	setupLogger(finalLogLevel)

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
	setupRoutes(app, m, yamlConfig)

	// Start server
	go func() {
		log.Info().Int("port", finalPort).Msg("Starting server")

		for _, list := range yamlConfig.Lists {
			log.Info().Str("id", list.ID).Str("desc", list.Description).Msg("Loaded mapping")
		}

		if err := app.Listen(fmt.Sprintf(":%d", finalPort)); err != nil {
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

func setupRoutes(app *fiber.App, m *mapper.Mapper, yamlConfig *config.MappingConfig) {
	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Transformation endpoint
	app.Post("/:map/query", handleTransform(m))

	// Kalamar plugin endpoint
	app.Get("/", handleKalamarPlugin(yamlConfig))
	app.Get("/:map", handleKalamarPlugin(yamlConfig))
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

		// Parse direction
		direction, err := mapper.ParseDirection(dir)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Apply mappings
		result, err := m.ApplyQueryMappings(mapID, mapper.MappingOptions{
			Direction: direction,
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

func handleKalamarPlugin(yamlConfig *config.MappingConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		mapID := c.Params("map")

		// Get list of available mappings
		var mappings []TemplateMapping
		for _, list := range yamlConfig.Lists {
			mappings = append(mappings, TemplateMapping{
				ID:          list.ID,
				Description: list.Description,
			})
		}

		// Use values from config (defaults are already applied during parsing)
		server := yamlConfig.Server
		sdk := yamlConfig.SDK

		// Prepare template data
		data := TemplateData{
			Title:       config.Title,
			Version:     config.Version,
			Hash:        config.Buildhash,
			Date:        config.Buildtime,
			Description: config.Description,
			Server:      server,
			SDK:         sdk,
			MapID:       mapID,
			Mappings:    mappings,
		}

		// Generate HTML
		html := generateKalamarPluginHTML(data)

		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	}
}

// generateKalamarPluginHTML creates the HTML template for the Kalamar plugin page
// This function can be easily modified to change the appearance and content
func generateKalamarPluginHTML(data TemplateData) string {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>` + data.Title + `</title>
    <script src="` + data.SDK + `"
            data-server="` + data.Server + `"></script>
</head>
<body>
    <div class="container">
        <h1>` + data.Title + `</h1>
		<p>` + data.Description + `</p>`

	if data.MapID != "" {
		html += `<p>Map ID: ` + data.MapID + `</p>`
	}

	html += `		<h2>Plugin Information</h2>
        <p><strong>Version:</strong> <tt>` + data.Version + `</tt></p>
		<p><strong>Build Date:</strong> <tt>` + data.Date + `</tt></p>
		<p><strong>Build Hash:</strong> <tt>` + data.Hash + `</tt></p>

        <h2>Available API Endpoints</h2>
        <dl>

		    <dt><tt><strong>GET</strong> /:map</tt></dt>
            <dd><small>Kalamar integration</small></dd>

			<dt><tt><strong>POST</strong> /:map/query</tt></dt>
            <dd><small>Transform JSON query objects using term mapping rules</small></dd>
			
        </dl>
		
		<h2>Available Term Mappings</h2>
	    <dl>`

	for _, m := range data.Mappings {
		html += `<dt><tt>` + m.ID + `</tt></dt>`
		html += `<dd>` + m.Description + `</dd>`
	}

	html += `
    </dl>`

	if data.MapID != "" {
		html += `   <script>
  		<!-- activates/deactivates Mapper. -->
  		  
       let data = {
         'action'  : 'pipe',
         'service' : 'https://korap.ids-mannheim.de/plugin/termmapper/` + data.MapID + `/query'
       };

       function pluginit (p) {
         p.onMessage = function(msg) {
           if (msg.key == 'termmapper') {
             if (msg.value) {
               data['job'] = 'add';
             }
             else {
               data['job'] = 'del';
             };
             KorAPlugin.sendMsg(data);
           };
         };
       };
    </script>`
	}

	html += `  </body>
</html>`

	return html
}

// expandGlobs expands glob patterns in the slice of file paths
// Returns the expanded list of files or an error if glob expansion fails
func expandGlobs(patterns []string) ([]string, error) {
	var expanded []string

	for _, pattern := range patterns {
		// Use filepath.Glob which works cross-platform
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to expand glob pattern '%s': %w", pattern, err)
		}

		// If no matches found, treat as literal filename (consistent with shell behavior)
		if len(matches) == 0 {
			log.Warn().Str("pattern", pattern).Msg("Glob pattern matched no files, treating as literal filename")
			expanded = append(expanded, pattern)
		} else {
			expanded = append(expanded, matches...)
		}
	}

	return expanded, nil
}
