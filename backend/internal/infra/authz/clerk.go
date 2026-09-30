// Package authz は Clerk が発行したセッショントークン(JWT)の検証を行います。
//
// フロントは Clerkk Elements で自前構築したログインUIからサインインし、
// 取得したセッショントークンを Authorization: Bearer で送ってきます。
// パスワードなどの認証情報は一切扱いません。
package authz

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrNoToken      = errors.New("authz: トークンがありません")
	ErrInvalidToken = errors.New("authz: トークンが不正です")
)

// Claims は検証済みトークンから取り出した利用者情報です。
type Claims struct {
	// ClerkUserID は Clerk のユーザーID（JWTの sub）。
	ClerkUserID string
	Email       string
	DisplayName string
}

// Verifier は Clerk の JWKS を使ってトークンを検証します。
type Verifier struct {
	issuer     string
	audience   string
	httpClient *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	ttl       time.Duration
}

// NewVerifier 検証器を作ります。
// issuer は Clerk の Frontend API のURL（例: https://xxx.clerk.accounts.dev）。
func NewVerifier(issuer, audience string) *Verifier {
	return &Verifier{
		issuer:     strings.TrimRight(issuer, "/"),
		audience:   audience,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		keys:       map[string]*rsa.PublicKey{},
		ttl:        10 * time.Minute,
	}
}

// Verify はトークン文字列を検証し、利用者情報を返します。
func (v *Verifier) Verify(ctx context.Context, tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrNoToken
	}

	parsed, err := jwt.Parse(
		tokenString, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("authz: 想定外の署名方式です: %v", t.Header["alg"])
			}
			kid, _ := t.Header["kid"].(string)
			if kid == "" {
				return nil, errors.New("authz: kid がありません")
			}
			return v.keyByID(ctx, kid)
		},
		jwt.WithIssuer(v.issuer),
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !parsed.Valid {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	mc, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	if v.audience != "" {
		if aud, err := mc.GetAudience(); err != nil || !contains(aud, v.audience) {
			return nil, fmt.Errorf("%w: audience が一致しません", ErrInvalidToken)
		}
	}

	sub, _ := mc["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("%w: sub がありません", ErrInvalidToken)
	}

	c := &Claims{ClerkUserID: sub}
	// email / name は Clerk の JWT テンプレートに含めておく前提。
	// 未設定でも動くように、無ければ空のままにする。
	if s, ok := mc["email"].(string); ok {
		c.Email = s
	}
	if s, ok := mc["name"].(string); ok {
		c.DisplayName = s
	}
	return c, nil
}

// keyByID は JWKS から公開鍵を引きます（TTL付きキャッシュ）。
func (v *Verifier) keyByID(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := time.Since(v.fetchedAt) < v.ttl
	v.mu.RUnlock()
	if ok && fresh {
		return key, nil
	}

	if err := v.refresh(ctx); err != nil {
		// 取得に失敗しても、期限切れキャッシュが使えるなら使う
		v.mu.RLock()
		key, ok = v.keys[kid]
		v.mu.RUnlock()
		if ok {
			return key, nil
		}
		return nil, err
	}

	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("authz: 鍵が見つかりません kid=%s", kid)
	}
	return key, nil
}

type jwksDocument struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func (v *Verifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.issuer+"/.well-known/jwks.json", nil)
	if err != nil {
		return fmt.Errorf("authz: JWKS要求の作成に失敗しました: %w", err)
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("authz: JWKSの取得に失敗しました: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authz: JWKSが異常応答です status=%d", resp.StatusCode)
	}

	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("authz: JWKSの解析に失敗しました: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := rsaPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("authz: 利用できる鍵がありません")
	}

	v.mu.Lock()
	v.keys = keys
	v.fetchedAt = time.Now()
	v.mu.Unlock()
	return nil
}

func rsaPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}
	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	if e == 0 {
		return nil, errors.New("authz: 指数が不正です")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
