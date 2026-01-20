package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hmw/cos-uploader/config"
	"github.com/hmw/cos-uploader/logger"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// 初始化日志
	log := logger.NewLogger()
	defer log.Sync()

	log.Info("Starting COS uploader", "config", *configPath)

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	log.Info("Config loaded successfully", "projects", len(cfg.Projects))

	// 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Info("Shutting down...")
}
