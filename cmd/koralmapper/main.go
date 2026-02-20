package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	texttemplate "text/template"
	"time"

	"github.com/KorAP/Koral-Mapper/config"
	"github.com/KorAP/Koral-Mapper/mapper"
	"github.com/alecthomas/kong"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//go:embed static/*
var staticFS embed.FS

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

type BasePageData struct {
	Title       string
	Version     string
	Hash        string
	Date        string
	Description string
	Server      string
	SDK         string
	ServiceURL  string
}

type SingleMappingPageData struct {
	BasePageData
	MapID       string
	Mappings    []config.MappingList
	QueryURL    string
	ResponseURL string
}

type QueryParams struct {
	Dir      string
	FoundryA string
	FoundryB string
	LayerA   string
	LayerB   string
}

// requestParams holds common request parameters
type requestParams struct {
	MapID    string
	Dir      string
	FoundryA string
	FoundryB string
	LayerA   string
	LayerB   string
}

// ConfigPageData holds all data passed to the configuration page template.
type ConfigPageData struct {
	BasePageData
	AnnotationMappings []config.MappingList
	CorpusMappings     []config.MappingList
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

// setupFiberLogger configures fiber's logger middleware to integrate with zerolog
func setupFiberLogger() fiber.Handler {
	// Check if HTTP request logging should be enabled based on current log level
	currentLevel := zerolog.GlobalLevel()

	// Only enable HTTP request logging if log level is debug or info
	if currentLevel > zerolog.InfoLevel {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	return func(c *fiber.Ctx) error {
		// Record start time
		start := time.Now()

		// Process request
		err := c.Next()

		// Calculate latency
		latency := time.Since(start)
		status := c.Response().StatusCode()

		// Determine log level based on status code
		logEvent := log.Info()
		if status >= 400 && status < 500 {
			logEvent = log.Warn()
		} else if status >= 500 {
			logEvent = log.Error()
		}

		// Log the request
		logEvent.
			Int("status", status).
			Dur("latency", latency).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent")).
			Msg("HTTP request")

		return err
	}
}

// extractRequestParams extracts and validates common request parameters
func extractRequestParams(c *fiber.Ctx) (*requestParams, error) {
	params := &requestParams{
		MapID:    c.Params("map"),
		Dir:      c.Query("dir", "atob"),
		FoundryA: c.Query("foundryA", ""),
		FoundryB: c.Query("foundryB", ""),
		LayerA:   c.Query("layerA", ""),
		LayerB:   c.Query("layerB", ""),
	}

	// Validate input parameters
	if err := validateInput(params.MapID, params.Dir, params.FoundryA, params.FoundryB, params.LayerA, params.LayerB, c.Body()); err != nil {
		return nil, err
	}

	// Validate direction
	if params.Dir != "atob" && params.Dir != "btoa" {
		return nil, fmt.Errorf("invalid direction, must be 'atob' or 'btoa'")
	}

	return params, nil
}

// parseRequestBody parses JSON request body and direction
func parseRequestBody(c *fiber.Ctx, dir string) (any, mapper.Direction, error) {
	var jsonData any
	if err := c.BodyParser(&jsonData); err != nil {
		return nil, mapper.BtoA, fmt.Errorf("invalid JSON in request body")
	}

	direction, err := mapper.ParseDirection(dir)
	if err != nil {
		return nil, mapper.BtoA, err
	}

	return jsonData, direction, nil
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
		ReadBufferSize:        64 * 1024, // 64KB - increase header size limit
		WriteBufferSize:       64 * 1024, // 64KB - increase response buffer size
	})

	// Add zerolog-integrated logger middleware
	app.Use(setupFiberLogger())

	// Set up routes
	setupRoutes(app, m, yamlConfig)

	// Start server
	go func() {
		log.Info().Int("port", finalPort).Msg("Starting server")
		fmt.Printf("Starting server port=%d\n", finalPort)

		for _, list := range yamlConfig.Lists {
			log.Info().Str("id", list.ID).Str("desc", list.Description).Msg("Loaded mapping")
			fmt.Printf("Loaded mapping desc=%s id=%s\n",
				formatConsoleField(list.Description),
				list.ID,
			)
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
	configTmpl := template.Must(template.ParseFS(staticFS, "static/config.html"))
	pluginTmpl := texttemplate.Must(texttemplate.ParseFS(staticFS, "static/plugin.html"))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Static file serving from embedded FS
	app.Get("/static/*", handleStaticFile())

	// Composite cascade transformation endpoints
	app.Post("/query", handleCompositeQueryTransform(m, yamlConfig.Lists))
	app.Post("/response", handleCompositeResponseTransform(m, yamlConfig.Lists))

	// Transformation endpoint
	app.Post("/:map/query", handleTransform(m))

	// Response transformation endpoint
	app.Post("/:map/response", handleResponseTransform(m))

	// Kalamar plugin endpoint
	app.Get("/", handleKalamarPlugin(yamlConfig, configTmpl, pluginTmpl))
	app.Get("/:map", handleKalamarPlugin(yamlConfig, configTmpl, pluginTmpl))
}

func handleStaticFile() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("*")
		data, err := fs.ReadFile(staticFS, "static/"+name)
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("not found")
		}
		switch {
		case strings.HasSuffix(name, ".js"):
			c.Set("Content-Type", "text/javascript; charset=utf-8")
		case strings.HasSuffix(name, ".css"):
			c.Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(name, ".html"):
			c.Set("Content-Type", "text/html; charset=utf-8")
		}
		return c.Send(data)
	}
}

func buildBasePageData(yamlConfig *config.MappingConfig) BasePageData {
	return BasePageData{
		Title:       config.Title,
		Version:     config.Version,
		Hash:        config.Buildhash,
		Date:        config.Buildtime,
		Description: config.Description,
		Server:      yamlConfig.Server,
		SDK:         yamlConfig.SDK,
		ServiceURL:  yamlConfig.ServiceURL,
	}
}

func buildConfigPageData(yamlConfig *config.MappingConfig) ConfigPageData {
	data := ConfigPageData{
		BasePageData: buildBasePageData(yamlConfig),
	}

	for _, list := range yamlConfig.Lists {
		normalized := list
		if normalized.Type == "" {
			normalized.Type = "annotation"
		}
		if list.IsCorpus() {
			data.CorpusMappings = append(data.CorpusMappings, normalized)
		} else {
			data.AnnotationMappings = append(data.AnnotationMappings, normalized)
		}
	}
	return data
}

func handleCompositeQueryTransform(m *mapper.Mapper, lists []config.MappingList) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cfgRaw := c.Query("cfg", "")
		if len(cfgRaw) > maxParamLength {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("cfg too long (max %d bytes)", maxParamLength),
			})
		}

		var jsonData any
		if err := c.BodyParser(&jsonData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON in request body",
			})
		}

		entries, err := ParseCfgParam(cfgRaw, lists)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		if len(entries) == 0 {
			return c.JSON(jsonData)
		}

		orderedIDs := make([]string, 0, len(entries))
		opts := make([]mapper.MappingOptions, 0, len(entries))
		for _, entry := range entries {
			dir := mapper.AtoB
			if entry.Direction == "btoa" {
				dir = mapper.BtoA
			}

			orderedIDs = append(orderedIDs, entry.ID)
			opts = append(opts, mapper.MappingOptions{
				Direction: dir,
				FoundryA:  entry.FoundryA,
				LayerA:    entry.LayerA,
				FoundryB:  entry.FoundryB,
				LayerB:    entry.LayerB,
			})
		}

		result, err := m.CascadeQueryMappings(orderedIDs, opts, jsonData)
		if err != nil {
			log.Error().Err(err).Str("cfg", cfgRaw).Msg("Failed to apply composite query mappings")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(result)
	}
}

func handleCompositeResponseTransform(m *mapper.Mapper, lists []config.MappingList) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cfgRaw := c.Query("cfg", "")
		if len(cfgRaw) > maxParamLength {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("cfg too long (max %d bytes)", maxParamLength),
			})
		}

		var jsonData any
		if err := c.BodyParser(&jsonData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON in request body",
			})
		}

		entries, err := ParseCfgParam(cfgRaw, lists)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		if len(entries) == 0 {
			return c.JSON(jsonData)
		}

		orderedIDs := make([]string, 0, len(entries))
		opts := make([]mapper.MappingOptions, 0, len(entries))
		for _, entry := range entries {
			dir := mapper.AtoB
			if entry.Direction == "btoa" {
				dir = mapper.BtoA
			}

			orderedIDs = append(orderedIDs, entry.ID)
			opts = append(opts, mapper.MappingOptions{
				Direction: dir,
				FoundryA:  entry.FoundryA,
				LayerA:    entry.LayerA,
				FoundryB:  entry.FoundryB,
				LayerB:    entry.LayerB,
			})
		}

		result, err := m.CascadeResponseMappings(orderedIDs, opts, jsonData)
		if err != nil {
			log.Error().Err(err).Str("cfg", cfgRaw).Msg("Failed to apply composite response mappings")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(result)
	}
}

func handleTransform(m *mapper.Mapper) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract and validate parameters
		params, err := extractRequestParams(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Parse request body
		jsonData, direction, err := parseRequestBody(c, params.Dir)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Apply mappings
		result, err := m.ApplyQueryMappings(params.MapID, mapper.MappingOptions{
			Direction: direction,
			FoundryA:  params.FoundryA,
			FoundryB:  params.FoundryB,
			LayerA:    params.LayerA,
			LayerB:    params.LayerB,
		}, jsonData)

		if err != nil {
			log.Error().Err(err).
				Str("mapID", params.MapID).
				Str("direction", params.Dir).
				Msg("Failed to apply mappings")

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(result)
	}
}

func handleResponseTransform(m *mapper.Mapper) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract and validate parameters
		params, err := extractRequestParams(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Parse request body
		jsonData, direction, err := parseRequestBody(c, params.Dir)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Apply response mappings
		result, err := m.ApplyResponseMappings(params.MapID, mapper.MappingOptions{
			Direction: direction,
			FoundryA:  params.FoundryA,
			FoundryB:  params.FoundryB,
			LayerA:    params.LayerA,
			LayerB:    params.LayerB,
		}, jsonData)

		if err != nil {
			log.Error().Err(err).
				Str("mapID", params.MapID).
				Str("direction", params.Dir).
				Msg("Failed to apply response mappings")

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
		// Check input lengths and invalid characters in one combined condition
		if len(param.value) > maxParamLength {
			return fmt.Errorf("%s too long (max %d bytes)", param.name, maxParamLength)
		}
		if strings.ContainsAny(param.value, "<>{}[]\\") {
			return fmt.Errorf("%s contains invalid characters", param.name)
		}
	}

	if len(body) > maxInputLength {
		return fmt.Errorf("request body too large (max %d bytes)", maxInputLength)
	}

	return nil
}

func handleKalamarPlugin(yamlConfig *config.MappingConfig, configTmpl *template.Template, pluginTmpl *texttemplate.Template) fiber.Handler {
	return func(c *fiber.Ctx) error {
		mapID := c.Params("map")

		// Config page (GET /)
		if mapID == "" {
			data := buildConfigPageData(yamlConfig)
			var buf bytes.Buffer
			if err := configTmpl.Execute(&buf, data); err != nil {
				log.Error().Err(err).Msg("Failed to execute config template")
				return c.Status(fiber.StatusInternalServerError).SendString("internal error")
			}
			c.Set("Content-Type", "text/html")
			return c.Send(buf.Bytes())
		}

		// Single-mapping page (GET /:map) — existing behavior
		// Get query parameters
		dir := c.Query("dir", "atob")
		foundryA := c.Query("foundryA", "")
		foundryB := c.Query("foundryB", "")
		layerA := c.Query("layerA", "")
		layerB := c.Query("layerB", "")

		// Validate input parameters and direction in one step
		if err := validateInput(mapID, dir, foundryA, foundryB, layerA, layerB, []byte{}); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		if dir != "atob" && dir != "btoa" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid direction, must be 'atob' or 'btoa'",
			})
		}

		queryParams := QueryParams{
			Dir:      dir,
			FoundryA: foundryA,
			FoundryB: foundryB,
			LayerA:   layerA,
			LayerB:   layerB,
		}

		queryURL, err := buildMapServiceURL(yamlConfig.ServiceURL, mapID, "query", queryParams)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to build query service URL")
			return c.Status(fiber.StatusInternalServerError).SendString("internal error")
		}
		reversed := queryParams
		if queryParams.Dir == "btoa" {
			reversed.Dir = "atob"
		} else {
			reversed.Dir = "btoa"
		}
		responseURL, err := buildMapServiceURL(yamlConfig.ServiceURL, mapID, "response", reversed)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to build response service URL")
			return c.Status(fiber.StatusInternalServerError).SendString("internal error")
		}

		data := SingleMappingPageData{
			BasePageData: buildBasePageData(yamlConfig),
			MapID:        mapID,
			Mappings:     yamlConfig.Lists,
			QueryURL:     queryURL,
			ResponseURL:  responseURL,
		}

		var buf bytes.Buffer
		if err := pluginTmpl.Execute(&buf, data); err != nil {
			log.Error().Err(err).Msg("Failed to execute plugin template")
			return c.Status(fiber.StatusInternalServerError).SendString("internal error")
		}
		c.Set("Content-Type", "text/html")
		return c.Send(buf.Bytes())
	}
}

func buildMapServiceURL(serviceURL, mapID, endpoint string, params QueryParams) (string, error) {
	service, err := url.Parse(serviceURL)
	if err != nil {
		return "", err
	}
	service.Path = path.Join(service.Path, mapID, endpoint)
	service.RawQuery = buildQueryParams(params.Dir, params.FoundryA, params.FoundryB, params.LayerA, params.LayerB)
	return service.String(), nil
}

func formatConsoleField(value string) string {
	if strings.ContainsAny(value, " \t") {
		return strconv.Quote(value)
	}
	return value
}

// buildQueryParams builds a query string from the provided parameters
func buildQueryParams(dir, foundryA, foundryB, layerA, layerB string) string {
	params := url.Values{}
	if dir != "" {
		params.Add("dir", dir)
	}
	if foundryA != "" {
		params.Add("foundryA", foundryA)
	}
	if foundryB != "" {
		params.Add("foundryB", foundryB)
	}
	if layerA != "" {
		params.Add("layerA", layerA)
	}
	if layerB != "" {
		params.Add("layerB", layerB)
	}
	return params.Encode()
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
