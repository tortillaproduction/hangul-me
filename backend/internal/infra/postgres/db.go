// Package postgres は NeonDB(PostgresSQL)上のリポジトリ実装です。
//
// ドライバは database/sql + lib/pq を使っています。
// lib/pq は外部依存を持たないため、ビルド環境を選びません。
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// scannable は *sql.Row と *sql.Rows の共通部分です。
type scannable interface {
	Scan(dest ...any) error
}

// Open は接続プールを作ります。
// NeonDB はサーバーレスで接続が切れやすいため、
// 接続寿命とアイドル時間を短めに設定しています。
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: 接続の初期化に失敗しました: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres: 接続確認に失敗しました: %w", err)
	}
	return db, nil
}
