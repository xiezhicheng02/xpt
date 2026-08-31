// Package migrate 内嵌 dht-service 的数据库迁移脚本。
package migrate

import (
	"database/sql/driver"
	"embed"
	"errors"
)

//go:embed *.sql
var FS embed.FS

type Hash [20]byte

// 实现 sql.Scanner 接口，支持从数据库扫描
func (n *Hash) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok || len(b) != 20 {
		return errors.New("invalid node id length")
	}
	copy(n[:], b)
	return nil
}

// 实现 driver.Valuer 接口，支持写入数据库
func (n Hash) Value() (driver.Value, error) {
	return n[:], nil
}
