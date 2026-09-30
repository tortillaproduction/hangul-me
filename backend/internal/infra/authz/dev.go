package authz

import (
	"context"
	"strings"
)

// DevVerifier は Clerk を設定していないローカル開発用の検証器です。
//
// Bearer トークンをそのまま Clerk ユーザーIDとして扱います。
// CLERK_ISSUER が設定されている本番・ステージングでは使われません。
type DevVerifier struct{}

func NewDevVerifier() *DevVerifier { return &DevVerifier{} }

func (d *DevVerifier) Verify(_ context.Context, token string) (*Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrNoToken
	}

	return &Claims{
		ClerkUserID: "dev_" + token,
		Email:       token + "@example.test",
		DisplayName: "開発ユーザー",
	}, nil
}
