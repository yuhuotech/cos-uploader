package uploader

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/hmw/cos-uploader/config"
	"github.com/hmw/cos-uploader/logger"
	cos "github.com/tencentyun/cos-go-sdk-v5"
)

// Uploader COS上传器
type Uploader struct {
	clients map[string]*cos.Client // project name -> COS client
	configs map[string]config.ProjectConfig
	queue   *Queue
	pool    *WorkerPool
	logger  *logger.Logger
	done    chan struct{}
	wg      sync.WaitGroup
}

// NewUploader 创建新的上传器
func NewUploader(projects []config.ProjectConfig, log *logger.Logger) (*Uploader, error) {
	u := &Uploader{
		clients: make(map[string]*cos.Client),
		configs: make(map[string]config.ProjectConfig),
		queue:   NewQueue(1000),
		logger:  log,
		done:    make(chan struct{}),
	}

	// 初始化每个项目的COS客户端
	for _, proj := range projects {
		client, err := createCOSClient(&proj.COSConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create COS client for project %s: %w", proj.Name, err)
		}
		u.clients[proj.Name] = client
		u.configs[proj.Name] = proj
		log.Info("COS client created", "project", proj.Name, "bucket", proj.COSConfig.Bucket)
	}

	// 创建工作池
	poolSize := 5
	if len(projects) > 0 {
		poolSize = projects[0].Watcher.PoolSize
	}
	u.pool = NewWorkerPool(poolSize, u, log)

	return u, nil
}

// createCOSClient 创建COS客户端
func createCOSClient(cosConfig *config.COSConfig) (*cos.Client, error) {
	// 构建COS URL
	urlStr := fmt.Sprintf("https://%s.cos.%s.myqcloud.com", cosConfig.Bucket, cosConfig.Region)
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse COS URL: %w", err)
	}

	// 创建授权传输
	authTransport := &cos.AuthorizationTransport{
		SecretID:  cosConfig.SecretID,
		SecretKey: cosConfig.SecretKey,
	}

	// 创建HTTP客户端
	httpClient := &http.Client{
		Transport: authTransport,
	}

	// 创建COS客户端
	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, httpClient)

	return client, nil
}

// Start 启动上传器
func (u *Uploader) Start() {
	u.wg.Add(1)
	go u.run()
	u.pool.Start()
}

// run 主循环
func (u *Uploader) run() {
	defer u.wg.Done()

	for {
		select {
		case <-u.done:
			return
		case task := <-u.queue.Tasks():
			if task == nil {
				return
			}
			u.pool.AddTask(task)
		}
	}
}

// AddTask 添加上传任务
func (u *Uploader) AddTask(task *UploadTask) {
	u.queue.Add(task)
}

// UploadFile 上传单个文件（由工作池调用）
func (u *Uploader) UploadFile(task *UploadTask) error {
	client, ok := u.clients[task.ProjectName]
	if !ok {
		return fmt.Errorf("COS client not found for project %s", task.ProjectName)
	}

	// 打开文件
	file, err := os.Open(task.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", task.FilePath, err)
	}
	defer file.Close()

	// 上传文件
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = client.Object.Put(ctx, task.RemotePath, file, nil)
	if err != nil {
		return fmt.Errorf("failed to upload file to COS: %w", err)
	}

	u.logger.Info("File uploaded successfully", "file", task.FilePath, "remote", task.RemotePath)
	return nil
}

// Stop 关闭上传器
func (u *Uploader) Stop() {
	close(u.done)
	u.pool.Stop()
	u.queue.Close()
	u.wg.Wait()
}

// WorkerPool 工作池
type WorkerPool struct {
	workers  int
	tasks    chan *UploadTask
	uploader *Uploader
	logger   *logger.Logger
	wg       sync.WaitGroup
	done     chan struct{}
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workers int, uploader *Uploader, log *logger.Logger) *WorkerPool {
	return &WorkerPool{
		workers:  workers,
		tasks:    make(chan *UploadTask, workers*2),
		uploader: uploader,
		logger:   log,
		done:     make(chan struct{}),
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	wp.logger.Info("Worker pool started", "workers", wp.workers)
}

// worker 工作协程
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.done:
			return
		case task := <-wp.tasks:
			if task == nil {
				return
			}

			wp.logger.Debug("Processing upload task", "worker", id, "file", task.FilePath)
			err := wp.uploader.UploadFile(task)
			if err != nil {
				// 重试逻辑
				if task.Retry < 3 {
					task.Retry++
					wp.logger.Warn("Upload failed, retrying", "file", task.FilePath, "retry", task.Retry, "error", err)
					wp.tasks <- task
				} else {
					// 3次都失败，记录日志
					wp.logger.Error("Upload failed after 3 retries", "file", task.FilePath, "error", err)
				}
			}
		}
	}
}

// AddTask 添加任务到工作池
func (wp *WorkerPool) AddTask(task *UploadTask) {
	wp.tasks <- task
}

// Stop 关闭工作池
func (wp *WorkerPool) Stop() {
	close(wp.done)
	close(wp.tasks)
	wp.wg.Wait()
}

// FullUploadStats 全量上传统计信息
type FullUploadStats struct {
	ProjectName      string
	TotalFiles       int64
	UploadedFiles    int64
	SkippedFiles     int64
	FailedFiles      int64
	TotalSize        int64
	UploadedSize     int64
	Duration         time.Duration
}

// ExecuteFullUpload 执行全量上传
// 返回统计信息和任何致命错误
func (u *Uploader) ExecuteFullUpload(projectName string) (*FullUploadStats, error) {
	startTime := time.Now()
	stats := &FullUploadStats{
		ProjectName: projectName,
	}

	// 获取项目配置
	projectConfig, ok := u.configs[projectName]
	if !ok {
		return nil, fmt.Errorf("project '%s' not found", projectName)
	}

	u.logger.Info("Starting full upload", "project", projectName)

	// 创建索引管理器
	cosClient, ok := u.clients[projectName]
	if !ok {
		return nil, fmt.Errorf("COS client not found for project '%s'", projectName)
	}
	indexManager := NewIndexManager(cosClient, &projectConfig.COSConfig, u.logger)

	// 创建扫描器
	scanner := NewDirectoryScanner(projectConfig, indexManager, u.logger)

	// Step 1: 扫描本地目录，生成本地索引
	u.logger.Info("Step 1: Scanning local directories", "project", projectName)
	localIdx, err := scanner.ScanDirectories()
	if err != nil {
		return nil, fmt.Errorf("failed to scan directories: %w", err)
	}

	stats.TotalFiles = int64(len(localIdx.Files))
	for _, entry := range localIdx.Files {
		stats.TotalSize += entry.Size
	}

	// 保存本地索引
	localIndexPath := GetLocalIndexPath(projectName)
	if err := localIdx.SaveToFile(localIndexPath); err != nil {
		u.logger.Warn("Failed to save local index", "error", err)
	} else {
		u.logger.Info("Local index saved", "path", localIndexPath)
	}

	// Step 2: 下载远程索引
	u.logger.Info("Step 2: Downloading remote index", "project", projectName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	remoteIdx, err := indexManager.DownloadRemoteIndex(ctx, projectName)
	cancel()
	if err != nil {
		u.logger.Warn("Failed to download remote index, will proceed anyway", "error", err)
		remoteIdx = NewFileIndex()
	}

	// Step 3: 对比索引，确定需要上传的文件
	u.logger.Info("Step 3: Analyzing files for upload", "project", projectName)
	filesToUpload, skipped := scanner.AnalyzeForUpload(localIdx, remoteIdx)
	stats.SkippedFiles = skipped

	if len(filesToUpload) == 0 {
		u.logger.Info("No files need to be uploaded", "project", projectName, "skipped", skipped)
		stats.Duration = time.Since(startTime)
		return stats, nil
	}

	// Step 4: 上传需要的文件
	u.logger.Info("Step 4: Starting file uploads", "project", projectName, "count", len(filesToUpload))
	successCount := 0
	failureCount := 0

	for localPath, entry := range filesToUpload {
		task := &UploadTask{
			FilePath:    localPath,
			RemotePath:  entry.RemotePath,
			ProjectName: projectName,
			Retry:       0,
		}

		// 上传文件（同步，带重试）
		err := u.uploadFileWithRetry(task, 3)
		if err != nil {
			u.logger.Error("File upload failed", "file", localPath, "error", err)
			failureCount++
		} else {
			successCount++
			stats.UploadedSize += entry.Size
			// 更新本地索引为已上传状态
			localIdx.Files[localPath].UploadedTime = time.Now().UTC().Format(time.RFC3339)
		}
	}

	stats.UploadedFiles = int64(successCount)
	stats.FailedFiles = int64(failureCount)

	// Step 5: 更新远程索引
	u.logger.Info("Step 5: Updating remote index", "project", projectName)
	UpdateRemoteIndexWithUploads(remoteIdx, filesToUpload)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	err = indexManager.UploadRemoteIndex(ctx, remoteIdx, projectName)
	cancel()
	if err != nil {
		u.logger.Warn("Failed to upload remote index", "error", err)
	}

	// 再次保存本地索引（带上传状态）
	if err := localIdx.SaveToFile(localIndexPath); err != nil {
		u.logger.Warn("Failed to save updated local index", "error", err)
	}

	stats.Duration = time.Since(startTime)

	// 输出统计报告
	u.logger.Info("Full upload completed",
		"project", projectName,
		"total_files", stats.TotalFiles,
		"uploaded", stats.UploadedFiles,
		"skipped", stats.SkippedFiles,
		"failed", stats.FailedFiles,
		"total_size", FormatBytes(stats.TotalSize),
		"uploaded_size", FormatBytes(stats.UploadedSize),
		"duration", stats.Duration.String())

	return stats, nil
}

// uploadFileWithRetry 上传文件并重试指定次数
func (u *Uploader) uploadFileWithRetry(task *UploadTask, maxRetries int) error {
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := u.UploadFile(task)
		if err == nil {
			return nil
		}

		if attempt < maxRetries-1 {
			// 指数退避重试
			waitTime := time.Duration(1<<uint(attempt)) * time.Second
			u.logger.Warn("Upload failed, retrying",
				"file", task.FilePath,
				"attempt", attempt+1,
				"max_attempts", maxRetries,
				"wait", waitTime.String(),
				"error", err)
			time.Sleep(waitTime)
		}
	}

	return fmt.Errorf("upload failed after %d attempts", maxRetries)
}