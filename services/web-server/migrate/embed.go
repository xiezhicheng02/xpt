// Package migrate 内嵌 web-server 的数据库迁移脚本。
package migrate

import "embed"

//go:embed *.sql
var FS embed.FS
