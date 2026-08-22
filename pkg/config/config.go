package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config 是对 viper 的轻量封装，提供类型化的配置读取。
// 每个服务在 main 中调用 Load 加载自己的 config.yaml。
type Config struct {
	v *viper.Viper
}

// Load 从指定路径读取 YAML 配置并初始化 Config。
// path 例如 "./services/tracker-service/config.yaml"。
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	return &Config{v: v}, nil
}

// LoadAny 依次尝试多个候选路径，返回第一个存在的配置文件。
// 解决"工作目录不同导致相对路径失效"的问题：
// 命令行运行（cwd=项目根）与 VSCode 调试（cwd=cmd 目录）都能正确加载。
//
//	cfg, err := config.LoadAny(
//	    "./services/dht-service/config.yaml", // 项目根运行
//	    "../config.yaml",                     // 从 cmd 目录运行/调试
//	)
func LoadAny(candidates ...string) (*Config, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no config candidates given")
	}
	var lastErr error
	for _, path := range candidates {
		if _, err := os.Stat(path); err != nil {
			lastErr = err
			continue
		}
		return Load(path)
	}
	return nil, fmt.Errorf("config file not found in any candidate path %v: %w", candidates, lastErr)
}

// GetString 返回字符串配置项。
func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}

// GetInt 返回整数配置项。
func (c *Config) GetInt(key string) int {
	return c.v.GetInt(key)
}

// GetBool 返回布尔配置项。
func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

// GetStringSlice 返回字符串切片配置项。
func (c *Config) GetStringSlice(key string) []string {
	return c.v.GetStringSlice(key)
}

// IsDebug 便捷方法：读取 debug 开关，用于日志级别。
func (c *Config) IsDebug() bool {
	return c.GetBool("debug")
}
