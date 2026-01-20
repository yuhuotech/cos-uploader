package uploader

import (
	"testing"

	"github.com/hmw/cos-uploader/logger"
)

func TestNewQueue(t *testing.T) {
	queue := NewQueue(10)
	if queue == nil {
		t.Fatal("Expected queue, got nil")
	}
	queue.Close()
}

func TestQueueAddGet(t *testing.T) {
	queue := NewQueue(10)
	defer queue.Close()

	task := &UploadTask{
		FilePath:    "/tmp/test.txt",
		RemotePath:  "uploads/test.txt",
		ProjectName: "project1",
	}

	go func() {
		queue.Add(task)
	}()

	got := queue.Get()
	if got.FilePath != task.FilePath {
		t.Errorf("Expected filepath %s, got %s", task.FilePath, got.FilePath)
	}
}

func TestWorkerPool(t *testing.T) {
	log := &logger.Logger{}
	pool := NewWorkerPool(3, nil, log)
	if pool == nil {
		t.Fatal("Expected pool, got nil")
	}

	if pool.workers != 3 {
		t.Errorf("Expected 3 workers, got %d", pool.workers)
	}
}
