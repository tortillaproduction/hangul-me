package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/hangulme/hangul-me/backend/internal/usecase/lookup"
)

// SearchLogRepository は search_queries のリポジトリです。
// 「表示された候補」ではなく「選択された候補」だけを残します。
type SearchLogRepository struct{ db *sql.DB }

func NewSearchLogRepository(db *sql.DB) *SearchLogRepository {
	return &SearchLogRepository{
		db: db,
	}
}

var _ lookup.SearchLogRepository = (*SearchLogRepository)(nil)

func (r *SearchLogRepository) Create(ctx context.Context, l *lookup.SearchLog) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}

	const q = `
		INSERT INTO search_queries
			(id, user_id, raw_input, normalized_input, matched_via, selected_entry_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`

	_, err := r.db.ExecContext(ctx,
		q, l.ID, l.UserID, l.RawInput, l.NormalizedInput, l.MatchedVia, l.SelectedEntryID, nowOr(l.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("searchlog: 記録に失敗しました: %w", err)
	}
	return nil
}
