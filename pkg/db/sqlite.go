// Package db 提供 sqlite 数据库连接与迁移执行工具。
package db

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	// modernc.org/sqlite 是纯 Go 实现的 sqlite 驱动（无需 CGO），
	// 驱动名为 "sqlite"。比 mattn/go-sqlite3 的构建更简单，适合学习与跨平台。
	_ "modernc.org/sqlite"
)

// ResolveDBPath 从多个候选路径中选择"父目录存在"的第一个。
// 数据库文件本身可不存在（首次运行会自动创建），所以只看父目录。
//
// 解决"工作目录不同导致相对路径失效"的问题：命令行运行（cwd=项目根）
// 与 VSCode 调试（cwd=cmd 目录）都能找到正确的 data 目录。
//
//	dbPath := db.ResolveDBPath(
//	    "./data/dht.db",        // 从项目根运行
//	    "../../../data/dht.db", // 从 cmd 目录运行/调试（services/xxx/cmd -> 项目根）
//	)
//
// 若所有候选的父目录都不存在，会尝试创建第一个候选的父目录（如项目根 data/）。
func ResolveDBPath(candidates ...string) string {
	for _, p := range candidates {
		dir := filepath.Dir(p)
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return p
		}
	}
	// 全部不可用：尝试创建第一个候选的父目录，兜底返回它。
	if len(candidates) > 0 {
		dir := filepath.Dir(candidates[0])
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return candidates[0]
		}
		return candidates[0]
	}
	return "./data/app.db"
}

// NewSQLite 打开（必要时创建）sqlite 数据库。
// 使用 WAL 模式提升并发读写性能，开启外键约束。
//
// modernc.org/sqlite 的 DSN 使用 _pragma= 语法传递 PRAGMA 参数：
//
//	file:<path>?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)
func NewSQLite(dbPath string) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)",
		dbPath,
	)
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite %s: %w", dbPath, err)
	}
	return db, nil
}

// RunMigrations 按文件名顺序执行给定 FS 下的所有 .sql 文件。
//
// 用法：每个服务用 go:embed 嵌入自己的 migrate/ 目录，
// 然后把 embed.FS 传给本函数：
//
//	//go:embed migrate/*.sql
//	var migrateFS embed.FS
//	db.RunMigrations(sqlDB, migrateFS)
//
// 也支持 FS 根目录直接是迁移文件（如 migrate 包内嵌 `*.sql`）。
// 学习提示：这里用"文件顺序 + 已执行记录"实现极简迁移；
// 生产项目可换用 golang-migrate / goose 等成熟工具。
func RunMigrations(d *sqlx.DB, fsys fs.FS) error {
	// 优先读取 migrate/ 子目录；若不存在则使用 FS 根目录。
	dir := "migrate"
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		dir = "."
		entries, err = fs.ReadDir(fsys, dir)
		if err != nil {
			return fmt.Errorf("read migrate dir: %w", err)
		}
	}

	// 按文件名排序保证执行顺序（001_xxx, 002_xxx ...）。
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// 建迁移记录表（幂等）。
	if _, err := d.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		// 已执行过的跳过。
		var applied int
		err := d.Get(&applied, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, e.Name())
		if err != nil {
			return fmt.Errorf("check migration %s: %w", e.Name(), err)
		}
		if applied > 0 {
			continue
		}

		// 当 FS 根目录就是迁移目录时，dir 为 "."，直接拼接文件名即可。
		readPath := e.Name()
		if dir != "." {
			readPath = dir + "/" + e.Name()
		}
		sqlBytes, err := fs.ReadFile(fsys, readPath)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", e.Name(), err)
		}

		// sqlite 驱动不支持一条 Exec 执行多条语句，按分号拆分执行。
		for _, stmt := range splitStatements(string(sqlBytes)) {
			if _, err := d.Exec(stmt); err != nil {
				return fmt.Errorf("apply migration %s: %w", e.Name(), err)
			}
		}

		if _, err := d.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, e.Name()); err != nil {
			return fmt.Errorf("record migration %s: %w", e.Name(), err)
		}
	}
	return nil
}

// splitStatements 将 SQL 文本按行拆分，忽略空行与 -- 注释行。
func splitStatements(sqlText string) []string {
	var stmts []string
	var cur strings.Builder
	for _, line := range strings.Split(sqlText, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		cur.WriteString(line)
		cur.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			stmts = append(stmts, cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		stmts = append(stmts, cur.String())
	}
	return stmts
}
