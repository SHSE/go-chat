package main

import (
	"context"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"github.com/shse/go-chat/transport"
	"github.com/shse/go-chat/chat"
)

type Config struct {
	Port uint16 `default:"3000"`
}

func main() {
	logger, err := zap.NewProduction()

	if err != nil {
		log.Fatal(err.Error())
	}

	defer logger.Sync()

	var config Config

	err = envconfig.Process("chat", &config)

	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	http.Handle("/metrics", promhttp.Handler())

	go http.ListenAndServe("0.0.0.0:8080", nil)

	logger.Info("Profiler is on, http://localhost:8080/debug/pprof")
	logger.Info("Prometheus metrics are on, http://localhost:8080/metrics")

	address := fmt.Sprintf("0.0.0.0:%d", config.Port)

	signals := make(chan os.Signal, 1)

	signal.Notify(signals,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-signals
		logger.Info("Shutting down")
		cancel()
	}()

	server := transport.NewServer(logger, prometheus.DefaultRegisterer)
	app := chat.NewChat(server)
	server.Run(ctx, address, app)
}
