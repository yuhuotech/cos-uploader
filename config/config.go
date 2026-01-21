package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 全局配置
type Config struct {
	Projects []ProjectConfig `yaml:"projects"`
	LogPath  string          `yaml:"log_path"` // 日志文件路径，默认: logs/cos-uploader.log
}

// ProjectConfig 项目配置
type ProjectConfig struct {
	Name        string            `yaml:"name"`
	Directories []string          `yaml:"directories"` // 监控的本地目录
	COSConfig   COSConfig         `yaml:"cos"`
	Watcher     WatcherConfig     `yaml:"watcher"`
	Alert       AlertConfig       `yaml:"alert"`
}

// COSConfig COS云存储配置
type COSConfig struct {
	SecretID   string `yaml:"secret_id"`
	SecretKey  string `yaml:"secret_key"`
	Region     string `yaml:"region"`      // 默认: ap-shanghai
	Bucket     string `yaml:"bucket"`      // bucket名称
	PathPrefix string `yaml:"path_prefix"` // 上传路径前缀
}

// WatcherConfig 文件监听配置
type WatcherConfig struct {
	Events   []string `yaml:"events"`     // 监听的事件类型: create, write, remove, rename, chmod
	PoolSize int      `yaml:"pool_size"` // 上传工作池大小
}

// AlertConfig 报警配置
type AlertConfig struct {
	DingTalkWebhook string `yaml:"dingtalk_webhook"` // 钉钉webhook URL
	Enabled         bool   `yaml:"enabled"`          // 是否启用报警
}

// LoadConfig 从YAML文件加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 验证和填充默认值
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if len(c.Projects) == 0 {
		return fmt.Errorf("no projects configured")
	}

	for i := range c.Projects {
		proj := &c.Projects[i]
		if proj.Name == "" {
			return fmt.Errorf("project %d missing name", i)
		}
		if len(proj.Directories) == 0 {
			return fmt.Errorf("project '%s' has no directories", proj.Name)
		}
		if proj.COSConfig.Bucket == "" {
			return fmt.Errorf("project '%s' missing COS bucket", proj.Name)
		}
		if proj.COSConfig.SecretID == "" || proj.COSConfig.SecretKey == "" {
			return fmt.Errorf("project '%s' missing COS credentials", proj.Name)
		}

		// 设置默认值
		if proj.COSConfig.Region == "" {
			proj.COSConfig.Region = "ap-shanghai"
		}
		if proj.Watcher.PoolSize == 0 {
			proj.Watcher.PoolSize = 5
		}
		if len(proj.Watcher.Events) == 0 {
			proj.Watcher.Events = []string{"create", "write"}
		}
	}

	return nil
}
