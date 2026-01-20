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