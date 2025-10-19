package app

import (
	"context"
	"net/http"
	"os"

	"github.com/megadata-dev/go-snmp-olt-zte-c320/config"
	"github.com/megadata-dev/go-snmp-olt-zte-c320/internal/exporter"
	"github.com/megadata-dev/go-snmp-olt-zte-c320/internal/handler"
	"github.com/megadata-dev/go-snmp-olt-zte-c320/internal/repository"
	"github.com/megadata-dev/go-snmp-olt-zte-c320/internal/usecase"
	"github.com/megadata-dev/go-snmp-olt-zte-c320/internal/utils"
	"github.com/megadata-dev/go-snmp-olt-zte-c320/pkg/graceful"
	"github.com/megadata-dev/go-snmp-olt-zte-c320/pkg/redis"
	"github.com/megadata-dev/go-snmp-olt-zte-c320/pkg/snmp"
	"github.com/prometheus/client_golang/prometheus"
	rds "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// App represents the main application structure that holds the HTTP router
// and manages the application lifecycle including dependencies initialization
// and server startup.
type App struct {
	router http.Handler
}

// New creates and returns a new instance of the App with initialized dependencies.
// It prepares the application for startup but does not start the server.
func New() *App {
	return &App{}
}

// Start initializes the application components, sets up connections to external services
// (Redis and SNMP), and starts the HTTP server. It handles graceful shutdown on context
// cancellation and ensures proper cleanup of resources.
//
// Parameters:
//   - ctx: context.Context for cancellation and timeout propagation
//
// Returns:
//   - error: returns any error that occurs during application startup or shutdown
func (a *App) Start(ctx context.Context) error {

	// Get config path from APP_ENV environment variable
	configPath := utils.GetConfigPath(os.Getenv("APP_ENV"))

	// Load configuration file from config path
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load config")
	}

	// Initialize Redis client
	redisClient := redis.NewRedisClient(cfg)

	// Check Redis connection
	err = redisClient.Ping(ctx).Err()
	if err != nil {
		log.Error().Err(err).Msg("Failed to ping Redis server")
	} else {
		log.Info().Msg("Redis server successfully connected")
	}

	// Close Redis client
	defer func(redisClient *rds.Client) {
		err := redisClient.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close Redis client")
		}
	}(redisClient)

	// Initialize SNMP connection
	snmpConn, err := snmp.SetupSnmpConnection(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup SNMP connection")
	}

	// Check SNMP connection
	/*
		if SNMP Connection with wrong credentials in SNMP v3, return error is nil
		if SNMP Connection with wrong Port in SNMP v2 v2c, return error is nil
		if SNMP Connection with wrong community v2 v2c, return error is nil

		Connect creates and opens a socket. Because UDP is a connectionless protocol,
		you won't know if the remote host is responding until you send packets.
		Neither will you know if the host is regularly disappearing and reappearing.
	*/

	if snmpConn.Connect() != nil {
		log.Error().Err(err).Msg("Failed to connect to SNMP server")
	} else {
		log.Info().Msg("SNMP server successfully connected")
	}

	// Close SNMP connection after application shutdown
	defer func() {
		if err := snmpConn.Conn.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close SNMP connection")
		}
	}()

	// Initialize repository
	snmpRepo := repository.NewPonRepository(snmpConn.Target, snmpConn.Community, snmpConn.Port)
	redisRepo := repository.NewOnuRedisRepo(redisClient)

	// Initialize usecase
	onuUsecase := usecase.NewOnuUsecase(snmpRepo, redisRepo, cfg)

	// Initialize handler
	onuHandler := handler.NewOnuHandler(onuUsecase)

	// Initialize and register the Prometheus collector
	onuCollector := exporter.NewOnuCollector(onuUsecase)
	prometheus.MustRegister(onuCollector)

	// Initialize router
	a.router = loadRoutes(onuHandler)

	// Start server
	addr := "8081"
	server := &http.Server{
		Addr:    ":" + addr,
		Handler: a.router,
	}

	// Start server at given address
	log.Info().Msgf("Application started at %s", addr)

	// Graceful shutdown
	return graceful.Shutdown(ctx, server)
}
