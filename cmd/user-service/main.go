package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rx3lixir/user-service/internal/config"
	"github.com/rx3lixir/user-service/internal/db"
	pb "github.com/rx3lixir/user-service/user-grpc/gen/go"
	"google.golang.org/grpc"
)

func main() {
	// Настраиваем логирование
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Создаем контекст, который можно отменить при получении сигнала остановки
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем обработку сигналов для грациозного завершения
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalCh
		slog.Info("Shutting down gracefully...")
		cancel()
	}()

	c, err := config.New()
	if err != nil {
		slog.Error("error creating config file", "error", err)
		os.Exit(1)
	}

	// Создаем пул соединений с базой данных
	pool, err := db.CreatePostgresPool(ctx, c.DB.DSN())
	if err != nil {
		slog.Error("Failed to create postgres pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("Connected to database")

	// Создаем хранилище и gRPC сервер
	storer := db.NewPosgresStore(pool)
	srv := server.NewServer(storer)

	// Настраиваем gRPC сервер
	grpcServer := grpc.NewServer(
	// Здесь можно добавить перехватчики (interceptors) для логирования, трассировки и т.д.
	)
	pb.(grpcServer, srv)

	// Включаем reflection API для gRPC (полезно для отладки)
	reflection.Register(grpcServer)

	// Запускаем gRPC сервер
	listener, err := net.Listen("tcp", c.Server.Address)
	if err != nil {
		slog.Error("Failed to start listener", "error", err)
		os.Exit(1)
	}

	slog.Info("Server is listening", "address", c.Server.Address)

	// Запускаем сервер в горутине
	serverError := make(chan error, 1)
	go func() {
		serverError <- grpcServer.Serve(listener)
	}()

	// Ждем либо завершения контекста (по сигналу), либо ошибки сервера
	select {
	case <-ctx.Done():
		grpcServer.GracefulStop()
		slog.Info("Server stopped gracefully")
	case err := <-serverError:
		slog.Error("Server error", "error", err)
	}
}
