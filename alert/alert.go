package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hmw/cos-uploader/logger"
)

// Alert 报警器
type Alert struct {
	webhook string
	logger  *logger.Logger
	client  *http.Client
}

// DingTalkMessage 钉钉消息格式
type DingTalkMessage struct {
	MsgType string      `json:"msgtype"`
	Text    TextContent `json:"text"`
}

// TextContent 文本内容
type TextContent struct {
	Content string `json:"content"`
}

// NewAlert 创建报警器
func NewAlert(webhook string, log *logger.Logger) *Alert {
	return &Alert{
		webhook: webhook,
		logger:  log,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendAlert 发送报警
func (a *Alert) SendAlert(title, message string) error {
	if a.webhook == "" {
		a.logger.Warn("DingTalk webhook not configured, alert not sent", "title", title)
		return nil
	}

	// 构建消息
	content := fmt.Sprintf("%s\n\n%s\n\nTime: %s", title, message, time.Now().Format(time.RFC3339))
	msg := DingTalkMessage{
		MsgType: "text",
		Text: TextContent{
			Content: content,
		},
	}

	// 转换为JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 发送HTTP POST请求
	resp, err := a.client.Post(a.webhook, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send alert: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DingTalk API returned status %d: %s", resp.StatusCode, string(body))
	}

	a.logger.Info("Alert sent successfully", "title", title)
	return nil
}

// SendUploadFailureAlert 发送上传失败报警
func (a *Alert) SendUploadFailureAlert(projectName, filePath string, err error) error {
	title := "COS Upload Failed"
	message := fmt.Sprintf("Project: %s\nFile: %s\nError: %v", projectName, filePath, err)
	return a.SendAlert(title, message)
}
