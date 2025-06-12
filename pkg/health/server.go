package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rx3lixir/user-service/pkg/logger"
)

// Server структура для healthcheck сервера
type Server struct {
	config Config
	health *Health
	server *http.Server
	pool   *pgxpool.Pool
	log    logger.Logger
}

// NewServer создает новый healthcheck сервер
func NewServer(pool *pgxpool.Pool, log logger.Logger, opts ...Option) *Server {
	// Применяем дефолтную конфигурацию
	config := defaultConfig()

	// Применяем все переданные опции
	for _, opt := range opts {
		opt(&config)
	}

	// Создаем health checker с настройками из конфига
	healthChecker := New(
		config.ServiceName,
		config.Version,
		WithTimeout(config.Timeout),
	)

	s := &Server{
		config: config,
		health: healthChecker,
		pool:   pool,
		log:    log,
	}

	s.setupChecks()
	s.setupRoutes()

	return s
}

// setupChecks настраивает все проверки здоровья для микросервиса
func (s *Server) setupChecks() {
	// Проверка базы данных
	s.health.AddCheck("database", PostgresChecker(s.pool))

	// Проверка миграций
	s.health.AddCheck("migrations", MigrationChecker(s.pool, s.config.MigrationVersion))

	// Проверка обязательных таблиц
	if len(s.config.RequiredTables) > 0 {
		s.health.AddCheck("required_tables", SimpleTableChecker(s.pool, s.config.RequiredTables))
	}

	s.log.Info("Health checks configured",
		"service", s.config.ServiceName,
		"version", s.config.Version,
		"port", s.config.Port,
		"timeout", s.config.Timeout,
		"database_check", true,
		"migrations_check", true,
		"tables_check", len(s.config.RequiredTables) > 0,
		"required_tables", s.config.RequiredTables,
		"migration_version", s.config.MigrationVersion,
	)
}

// setupRoutes настраивает HTTP маршруты
func (s *Server) setupRoutes() {
	mux := http.NewServeMux()

	// Основные эндпоинты
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/live", s.liveHandler)
	mux.HandleFunc("/info", s.infoHandler)

	s.server = &http.Server{
		Addr:         s.config.Port,
		Handler:      mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}
}

// Handler возвращает HTTP handler для health эндпоинта
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := s.health.Check(r.Context())

	// Устанавливаем статус код
	statusCode := http.StatusOK
	if response.Status == StatusDown {
		statusCode = http.StatusServiceUnavailable
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// liveHandler простая проверка живости сервиса
func (s *Server) liveHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ALIVE"))
}

// infoHandler возвращает информацию о сервисе
func (s *Server) infoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := map[string]any{
		"service":    s.config.ServiceName,
		"version":    s.config.Version,
		"build_time": time.Now().Format(time.RFC3339),
		"go_version": runtime.Version(),
		"endpoints": map[string]string{
			"health": "/health",
			"live":   "/live",
			"info":   "/info",
		},
	}

	if len(s.config.RequiredTables) > 0 {
		info["required_tables"] = s.config.RequiredTables
	}

	json.NewEncoder(w).Encode(info)
}

// Start запускает healthcheck сервер
func (s *Server) Start() error {
	s.log.Info("Starting health check server",
		"address", s.server.Addr,
		"service", s.config.ServiceName,
		"version", s.config.Version,
	)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("health server error: %w", err)
	}
	return nil
}

// Shutdown грациозно останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down health check server")
	return s.server.Shutdown(ctx)
}

// IsHealthy возвращает true если все проверки проходят
func (s *Server) IsHealthy(ctx context.Context) bool {
	response := s.health.Check(ctx)
	return response.Status == StatusUp
}
