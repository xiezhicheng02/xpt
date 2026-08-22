package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 是对 viper 的轻量封装，提供类型化的配置读取。
// 每个服务在 main 中调用 Load 加载自己的 config.yaml。
type Config struct {
	v *viper.Viper
}

// Load 从指定路径读取 YAML 配置并初始化 Config。
// 建议配合 apppath 包传入项目根下的绝对路径，避免 cwd 依赖。
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	return &Config{v: v}, nil
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
