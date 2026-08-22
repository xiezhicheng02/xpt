package repository

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/xiezc/xpt/pkg/model"
)

// ErrUserNotFound 表示用户不存在。
var ErrUserNotFound = errors.New("user not found")

// UserRepo 负责 users 表的读写。
type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create 创建用户（username 唯一，重复时返回错误）。
// TODO(学习): 密码哈希应在 service 层完成后再入库。
func (r *UserRepo) Create(u *model.User) error {
	_, err := r.db.Exec(`
INSERT INTO users(username, password_hash, email, is_admin)
VALUES (?, ?, ?, ?)`,
		u.Username, u.PasswordHash, u.Email, u.IsAdmin)
	return err
}

// GetByUsername 按用户名查询用户。
func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	var u model.User
	err := r.db.Get(&u, `
SELECT id, username, password_hash, email, is_admin, created_at
FROM users WHERE username = ?`, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByID 按 ID 查询用户。
func (r *UserRepo) GetByID(id int64) (*model.User, error) {
	var u model.User
	err := r.db.Get(&u, `
SELECT id, username, password_hash, email, is_admin, created_at
FROM users WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SetAdmin 设置或取消用户的管理员权限。
func (r *UserRepo) SetAdmin(id int64, isAdmin bool) error {
	_, err := r.db.Exec(`UPDATE users SET is_admin = ? WHERE id = ?`, isAdmin, id)
	return err
}
