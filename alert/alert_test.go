package alert

import (
	"testing"

	"github.com/hmw/cos-uploader/logger"
)

func TestNewAlert(t *testing.T) {
	log := logger.NewLogger()
	alert := NewAlert("https://example.com/webhook", log)
	if alert == nil {
		t.Fatal("Expected alert, got nil")
	}
}

func TestSendAlertWithoutWebhook(t *testing.T) {
	log := logger.NewLogger()
	alert := NewAlert("", log)

	// 应该不返回错误，只是warn日志
	err := alert.SendAlert("Test", "Test message")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestMessageFormat(t *testing.T) {
	msg := DingTalkMessage{
		MsgType: "text",
		Text: TextContent{
			Content: "test content",
		},
	}

	if msg.MsgType != "text" {
		t.Errorf("Expected msgtype 'text', got %s", msg.MsgType)
	}
}

func TestSendUploadFailureAlert(t *testing.T) {
	log := logger.NewLogger()
	alert := NewAlert("", log)

	// 测试发送上传失败报警
	err := alert.SendUploadFailureAlert("test-project", "/path/to/file", nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
