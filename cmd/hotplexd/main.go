package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hrygo/hotplex"
	"github.com/hrygo/hotplex/chatapps"
	"github.com/hrygo/hotplex/internal/server"
	"github.com/hrygo/hotplex/provider"
	"github.com/joho/godotenv"
)

func main() {
	// 0. Load .env file
	if err := godotenv.Load(); err != nil {
		// It's okay if .env doesn't exist, we'll use environmental variables or defaults
		_ = err
	}

	// 1. Configure logging
	logLevel := slog.LevelInfo
	if val := os.Getenv("LOG_LEVEL"); val != "" {
		switch strings.ToUpper(val) {
		case "DEBUG":
			logLevel = slog.LevelDebug
		case "WARN":
			logLevel = slog.LevelWarn
		case "ERROR":
			logLevel = slog.LevelError
		}
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Info("Starting HotPlex Proxy Server...", "log_level", logLevel)

	// 2. Initialize HotPlex Core Engine
	idleTimeout := 30 * time.Minute
	if val := os.Getenv("IDLE_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			idleTimeout = d
		}
	}

	executionTimeout := 30 * time.Minute
	if val := os.Getenv("EXECUTION_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			executionTimeout = d
		}
	}

	// 2.1 Decide Provider
	providerType := provider.ProviderType(os.Getenv("HOTPLEX_PROVIDER_TYPE"))
	if providerType == "" {
		providerType = provider.ProviderTypeClaudeCode
	}

	providerBinary := os.Getenv("HOTPLEX_PROVIDER_BINARY")
	providerModel := os.Getenv("HOTPLEX_PROVIDER_MODEL")

	prv, err := provider.CreateProvider(provider.ProviderConfig{
		Type:         providerType,
		Enabled:      true,
		BinaryPath:   providerBinary,
		DefaultModel: providerModel,
	})
	if err != nil {
		logger.Error("Failed to create provider", "type", providerType, "error", err)
		os.Exit(1)
	}

	// Load API key for admin operations
	adminToken := os.Getenv("HOTPLEX_API_KEY")
	if keys := os.Getenv("HOTPLEX_API_KEYS"); keys != "" {
		adminToken = strings.Split(keys, ",")[0]
	}

	// Warn if admin token is not configured
	if adminToken == "" {
		logger.Warn("SECURITY WARNING: No admin token configured. " +
			"Bypass mode will be unavailable. " +
			"Set HOTPLEX_API_KEY or HOTPLEX_API_KEYS environment variable for production use.")
	} else {
		logger.Info("Admin token configured", "token_length", len(adminToken))
	}

	opts := hotplex.EngineOptions{
		Timeout:     executionTimeout,
		IdleTimeout: idleTimeout,
		Logger:      logger,
		AdminToken:  adminToken,
		Provider:    prv,
	}

	engine, err := hotplex.NewEngine(opts)
	if err != nil {
		logger.Error("Failed to initialize HotPlex engine", "error", err)
		os.Exit(1)
	}

	// 2. Initialize CORS configuration and WebSocket handler
	corsConfig := server.NewSecurityConfig(logger)
	wsHandler := server.NewHotPlexWSHandler(engine, logger, corsConfig)
	http.Handle("/ws/v1/agent", wsHandler)

	// 2.1 Initialize OpenCode compatibility server
	if os.Getenv("HOTPLEX_OPENCODE_COMPAT_ENABLED") != "false" {
		openCodeSrv := server.NewOpenCodeHTTPHandler(engine, logger, corsConfig)
		ocRouter := mux.NewRouter()
		openCodeSrv.RegisterRoutes(ocRouter)
		http.Handle("/global/", ocRouter)
		http.Handle("/session", ocRouter)
		http.Handle("/session/", ocRouter)
		http.Handle("/config", ocRouter)
		logger.Info("OpenCode compatibility server initialized")
	}

	// 2.2 Initialize Observability handlers
	healthHandler := server.NewHealthHandler()
	metricsHandler := server.NewMetricsHandler()
	readyHandler := server.NewReadyHandler(func() bool { return engine != nil })
	liveHandler := server.NewLiveHandler()

	http.Handle("/health", healthHandler)
	http.Handle("/health/ready", readyHandler)
	http.Handle("/health/live", liveHandler)
	http.Handle("/metrics", metricsHandler)

	// 3. Initialize ChatApps adapters
	chatappsEnabled := os.Getenv("CHATAPPS_ENABLED")
	var chatappsMgr *chatapps.AdapterManager
	if chatappsEnabled == "true" {
		var chatappsHandler http.Handler
		var err error
		chatappsHandler, chatappsMgr, err = chatapps.Setup(context.Background(), logger)
		if err != nil {
			logger.Error("Failed to setup chatapps", "error", err)
		} else {
			http.Handle("/webhook/", chatappsHandler)
			logger.Info("ChatApps adapters initialized and webhooks registered")
		}
	}

	// Cleanup safety net (deferred immediately after engines/mgrs are ready)
	defer func() {
		logger.Info("Executing final cleanup safety net...")
		if chatappsMgr != nil {
			if err := chatappsMgr.StopAll(); err != nil {
				logger.Error("ChatApps cleanup failed", "error", err)
			}
		}
		if engine != nil {
			if err := engine.Close(); err != nil {
				logger.Error("Core engine cleanup failed", "error", err)
			}
		}
	}()

	// 4. Start HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: http.DefaultServeMux,
	}

	go func() {
		logger.Info("Listening on", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
			stop <- syscall.SIGTERM
		}
	}()

	<-stop
	logger.Info("Shutting down gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown failed", "error", err)
	}
}
