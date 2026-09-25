// Package analytics は登録単語の可視化に使う集計モデルです
//
// 集計の実体はSQL側（infra/postgres）にありますが、
// 「どの軸で見せるか」はドメインの関心事としてここに定義します
package analytics

import (
	"context"

	"github.com/google/uuid"
)

// RankedWord は「よく検索された」「よく見返した」単語の1行です
type RankedWord struct {
	EntryID     uuid.UUID
	Hangul      string
	ReadingKana string
	MeaningJA   string
	Count       int
}

// ConsonatBucket は初声・パッチムごとの登録数です
// 「自分がどの音に偏ってるか/つまずいてるか」を見るための軸
type ConsonantBucket struct {
	// Consonant は互換字母（ㄱ, ㄴ, …）
	// パッチム無しは空文字
	Consonant string
	Count     int
}

// TitleBucket は「どこで聞いた？」別の登録です
type TitleBucket struct {
	// Title が空文字なら未記入をまとめたもの
	Title string
	Count int
}

// DailyCount は日別の件数です（登録数の推移など）
type DailyCount struct {
	Date  string // YYYY-MM-DD
	Count int
}

// Overview はダッシュボードの分析ビューに必要な集計一式です
type Overview struct {
	TotalWords         int
	RegisteredThisWeek int
	MostSearched       []RankedWord
	MostViewed         []RankedWord
	InitialConsonants  []ConsonantBucket
	FinalConsonants    []ConsonantBucket
	EncounteredTitles  []TitleBucket
	RegistrationTrend  []DailyCount
}

// Repository は集計の取得境界です
type Repository interface {
	// OverviewForUser はユーザー自身の集計一式を返します
	OverviewForUser(ctx context.Context, userID uuid.UUID) (*Overview, error)

	// GlobalMostSearched はアプリ全体でよく選択された単語トップNを返します
	GlobalMostSearched(ctx context.Context, limit int) ([]RankedWord, error)
}
