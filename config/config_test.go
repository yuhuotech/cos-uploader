package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// 创建临时配置文件
	content := `projects:
  - name: project1
    directories:
      - /path/to/dir1
    cos:
      secret_id: test_id
      secret_key: test_key
      bucket: test-bucket
    watcher:
      pool_size: 5
    alert:
      dingtalk_webhook: https://oapi.dingtalk.com/robot/send?access_token=xxx
      enabled: true
`
	tmpFile, err := os.CreateTemp("", "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(cfg.Projects))
	}

	proj := cfg.Projects[0]
	if proj.Name != "project1" {
		t.Errorf("Expected project name 'project1', got '%s'", proj.Name)
	}

	if proj.COSConfig.Bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", proj.COSConfig.Bucket)
	}

	if proj.COSConfig.Region != "ap-shanghai" {
		t.Errorf("Expected default region 'ap-shanghai', got '%s'", proj.COSConfig.Region)
	}

	if proj.Watcher.PoolSize != 5 {
		t.Errorf("Expected pool size 5, got %d", proj.Watcher.PoolSize)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "empty projects",
			config:  &Config{Projects: []ProjectConfig{}},
			wantErr: true,
		},
		{
			name: "missing bucket",
			config: &Config{
				Projects: []ProjectConfig{
					{
						Name: "test",
						Directories: []string{"/tmp"},
						COSConfig: COSConfig{
							SecretID:  "id",
							SecretKey: "key",
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
