package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rx3lixir/user-service/internal/config"
	"github.com/rx3lixir/user-service/internal/db"
	"github.com/rx3lixir/user-service/pkg/health"
	"github.com/rx3lixir/user-service/pkg/logger"
	pb "github.com/rx3lixir/user-service/user-grpc/gen/go"
	"github.com/rx3lixir/user-service/user-grpc/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Загрузка конфигурации
	c, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	// Инициализация логгера
	logger.Init(c.Service.Env)
	defer logger.Close()

	// Создаем экземпляр логгера для передачи компонентам
	log := logger.NewLogger()

	// Создаем контекст, который можно отменить при получении сигнала остановки
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем обработку сигналов для грациозного завершения
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// Логируем конфигурацию для отладки
	log.Info("Configuration loaded",
		"env", c.Service.Env,
		"db_host", c.DB.Host,
		"db_port", c.DB.Port,
		"db_name", c.DB.DBName,
		"server_address", c.Server.Address,
	)

	// Создаем пул соединений с базой данных
	pool, err := db.CreatePostgresPool(ctx, c.DB.DSN())
	if err != nil {
		log.Error("Failed to create postgres pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("Connected to database")

	// Создаем хранилище и gRPC сервер
	storer := db.NewPosgresStore(pool)
	srv := server.NewServer(storer, log)

	// Настраиваем gRPC сервер
	grpcServer := grpc.NewServer(
	// Здесь можно добавить перехватчики (interceptors) для логирования, трассировки и т.д.
	)
	pb.RegisterUserServiceServer(grpcServer, srv)

	// Включаем reflection API для gRPC (полезно для отладки)
	reflection.Register(grpcServer)

	// Запускаем gRPC сервер
	listener, err := net.Listen("tcp", c.Server.Address)
	if err != nil {
		log.Error("Failed to start listener", "error", err)
		os.Exit(1)
	}

	log.Info("gRPC server is listening", "address", c.Server.Address)

	// Создаем HealthCheck сервер
	healthServer := health.NewServer(pool, log,
		health.WithServiceName("user-service"),
		health.WithVersion("1.0.0"),
		health.WithPort(":8083"),
		health.WithTimeout(5*time.Second),
		health.WithRequiredTables("users"),
	)

	// Запускаем серверы
	errCh := make(chan error, 2)

	// Health check сервер
	go func() {
		errCh <- healthServer.Start()
	}()

	// gRPC сервер
	go func() {
		errCh <- grpcServer.Serve(listener)
	}()

	// Ждем завершения
	select {
	case <-signalCh:
		log.Info("Shutting down gracefully...")

		// Останавливаем серверы
		grpcServer.GracefulStop()
		if err := healthServer.Shutdown(context.Background()); err != nil {
			log.Error("Health server shutdown error", "error", err)
		}

	case err := <-errCh:
		log.Error("Server error", "error", err)

		grpcServer.GracefulStop()
		if err := healthServer.Shutdown(context.Background()); err != nil {
			log.Error("Health server shutdown error", "error", err)
		}
	}
}
