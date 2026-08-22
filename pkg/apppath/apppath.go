// Package apppath 提供项目根目录定位工具。
//
// 背景：main.go 里的配置文件、数据库文件都是相对路径，而"相对路径"
// 依赖进程的工作目录（cwd）——命令行运行是项目根，VSCode 调试是 cmd 目录，
// 导致同一段代码在不同场景下找不到文件。
//
// 统一方案：Go 项目的根目录一定包含 go.mod，所以从当前工作目录
// 向上逐级查找 go.mod，找到即视为项目根。之后所有路径都基于项目根
// 拼接成绝对路径，与 cwd 完全无关。
package apppath

import (
	"os"
	"path/filepath"
)

// Root 返回项目根目录的绝对路径。
// 依次尝试：
//  1. 从当前工作目录向上查找 go.mod；
//  2. 从可执行文件所在目录向上查找 go.mod（如从 bin/ 启动二进制时）。
//
// 若都找不到返回空字符串。
func Root() string {
	if wd, err := os.Getwd(); err == nil {
		if r := findUp(wd); r != "" {
			return r
		}
	}
	if exe, err := os.Executable(); err == nil {
		if r := findUp(filepath.Dir(exe)); r != "" {
			return r
		}
	}
	return ""
}

// findUp 从 start 目录向上逐级查找 go.mod。
func findUp(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir { // 已到文件系统根
			return ""
		}
		dir = parent
	}
}

// MustRoot 与 Root 相同，但在找不到项目根时 panic（main 启动期失败应直接退出）。
func MustRoot() string {
	if r := Root(); r != "" {
		return r
	}
	panic("cannot locate project root: go.mod not found in any parent directory")
}

// Config 返回项目根下 services/<name>/config.yaml 的绝对路径。
// 例如 Config("dht-service") -> <root>/services/dht-service/config.yaml
func Config(service string) string {
	return filepath.Join(MustRoot(), "services", service, "config.yaml")
}

// Data 返回项目根下 data/<file> 的绝对路径。
// 例如 Data("dht.db") -> <root>/data/dht.db
func Data(file string) string {
	return filepath.Join(MustRoot(), "data", file)
}
