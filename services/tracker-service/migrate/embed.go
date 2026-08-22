// Package migrate 内嵌 tracker-service 的数据库迁移脚本。
package migrate

import "embed"

//go:embed *.sql
var FS embed.FS
