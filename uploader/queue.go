package uploader

import "sync"

// UploadTask 上传任务
type UploadTask struct {
	FilePath    string // 本地文件路径
	RemotePath  string // 远程COS路径
	ProjectName string // 项目名称
	Retry       int    // 重试次数
}

// Queue 上传任务队列
type Queue struct {
	tasks chan *UploadTask
	mu    sync.RWMutex
}

// NewQueue 创建新的任务队列
func NewQueue(bufferSize int) *Queue {
	return &Queue{
		tasks: make(chan *UploadTask, bufferSize),
	}
}

// Add 添加任务到队列
func (q *Queue) Add(task *UploadTask) {
	q.tasks <- task
}

// Get 从队列获取任务
func (q *Queue) Get() *UploadTask {
	return <-q.tasks
}

// Tasks 返回任务通道
func (q *Queue) Tasks() <-chan *UploadTask {
	return q.tasks
}

// Close 关闭队列
func (q *Queue) Close() {
	close(q.tasks)
}
