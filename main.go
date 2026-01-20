package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/hmw/cos-uploader/alert"
	"github.com/hmw/cos-uploader/config"
	"github.com/hmw/cos-uploader/logger"
	uploaderModule "github.com/hmw/cos-uploader/uploader"
	"github.com/hmw/cos-uploader/watcher"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	version := flag.Bool("version", false, "Show version information")
	fullUpload := flag.String("full-upload", "", "Execute full upload for specified project")
	flag.Parse()

	// 如果指定了 --version，输出版本后退出
	if *version {
		println("COS Uploader", Version)
		os.Exit(0)
	}

	// 初始化日志
	log := logger.NewLogger()
	defer log.Sync()

	log.Info("Starting COS uploader", "version", Version, "config", *configPath)

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	log.Info("Config loaded successfully", "projects", len(cfg.Projects))

	// 创建上传器
	uploaderSvc, err := uploaderModule.NewUploader(cfg.Projects, log)
	if err != nil {
		log.Error("Failed to create uploader", "error", err)
		os.Exit(1)
	}

	// 如果指定了全量上传，执行后退出
	if *fullUpload != "" {
		log.Info("Executing full upload", "project", *fullUpload)
		stats, err := uploaderSvc.ExecuteFullUpload(*fullUpload)
		if err != nil {
			log.Error("Full upload failed", "project", *fullUpload, "error", err)
			os.Exit(1)
		}

		// 打印统计报告
		println("")
		println("=" + strings.Repeat("=", 78) + "=")
		println("Full Upload Report")
		println("=" + strings.Repeat("=", 78) + "=")
		println(fmt.Sprintf("Project:       %s", stats.ProjectName))
		println(fmt.Sprintf("Total Files:   %d", stats.TotalFiles))
		println(fmt.Sprintf("Uploaded:      %d", stats.UploadedFiles))
		println(fmt.Sprintf("Skipped:       %d", stats.SkippedFiles))
		println(fmt.Sprintf("Failed:        %d", stats.FailedFiles))
		println(fmt.Sprintf("Total Size:    %s", uploaderModule.FormatBytes(stats.TotalSize)))
		println(fmt.Sprintf("Upload Size:   %s", uploaderModule.FormatBytes(stats.UploadedSize)))
		println(fmt.Sprintf("Duration:      %s", stats.Duration.String()))
		println("=" + strings.Repeat("=", 78) + "=")

		os.Exit(0)
	}

	// 创建报警器
	alerts := make(map[string]*alert.Alert)
	for _, proj := range cfg.Projects {
		if proj.Alert.Enabled && proj.Alert.DingTalkWebhook != "" {
			alerts[proj.Name] = alert.NewAlert(proj.Alert.DingTalkWebhook, log)
		}
	}

	// 启动文件监听和上传
	watchers := make([]*watcher.Watcher, 0)
	watcherGroup := sync.WaitGroup{}

	for _, proj := range cfg.Projects {
		// 为每个项目创建文件监听器
		w, err := watcher.NewWatcher(proj.Directories, proj.Watcher.Events, log)
		if err != nil {
			log.Error("Failed to create watcher", "project", proj.Name, "error", err)
			continue
		}

		watchers = append(watchers, w)

		// 启动监听
		w.Start()

		watcherGroup.Add(1)
		go func(proj config.ProjectConfig, w *watcher.Watcher) {
			defer watcherGroup.Done()

			for event := range w.Events() {
				// 检查文件是否存在
				if _, err := os.Stat(event.FilePath); os.IsNotExist(err) {
					continue
				}

				// 计算远程路径
				remotePath := calculateRemotePath(event.FilePath, proj)

				// 创建上传任务
				task := &uploaderModule.UploadTask{
					FilePath:    event.FilePath,
					RemotePath:  remotePath,
					ProjectName: proj.Name,
					Retry:       0,
				}

				log.Info("Adding upload task", "project", proj.Name, "file", event.FilePath, "remote", remotePath)
				uploaderSvc.AddTask(task)
			}
		}(proj, w)
	}

	// 启动上传器
	uploaderSvc.Start()

	// 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info("COS uploader is running, press Ctrl+C to exit")
	<-sigChan

	log.Info("Shutting down...")

	// 关闭所有监听器
	for _, w := range watchers {
		w.Close()
	}

	// 等待所有监听器完成
	watcherGroup.Wait()

	// 关闭上传器
	uploaderSvc.Stop()

	log.Info("COS uploader stopped")
}

// calculateRemotePath 计算远程COS路径
func calculateRemotePath(localPath string, proj config.ProjectConfig) string {
	// 获取相对于监控目录的相对路径
	for _, dir := range proj.Directories {
		if strings.HasPrefix(localPath, dir) {
			relPath, _ := filepath.Rel(dir, localPath)
			return proj.COSConfig.PathPrefix + relPath
		}
	}
	return proj.COSConfig.PathPrefix + filepath.Base(localPath)
}
