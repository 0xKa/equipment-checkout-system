package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/0xKa/equipment-checkout-system/server/types"
	"github.com/coreos/go-oidc/v3/oidc"
)

const maxJWKSResponseBytes int64 = 1024 * 1024

type OIDCVerifierConfig struct {
	IssuerURL   string
	JWKSURL     string
	Audience    string
	HTTPTimeout time.Duration
	ClockSkew   time.Duration
}

type oidcTokenVerifier struct {
	verifier  *oidc.IDTokenVerifier
	audience  string
	clockSkew time.Duration
	now       func() time.Time
}

var _ TokenVerifier = (*oidcTokenVerifier)(nil)

// NewOIDCTokenVerifier checks JWKS availability and constructs one long-lived,
// cached remote-key verifier.
func NewOIDCTokenVerifier(
	ctx context.Context,
	cfg OIDCVerifierConfig,
) (TokenVerifier, error) {
	client := &http.Client{Timeout: cfg.HTTPTimeout}
	if err := checkJWKSAvailability(ctx, client, cfg.JWKSURL); err != nil {
		return nil, err
	}

	keySetContext := oidc.ClientContext(context.Background(), client)
	keySet := oidc.NewRemoteKeySet(keySetContext, cfg.JWKSURL)
	verifier := oidc.NewVerifier(cfg.IssuerURL, keySet, &oidc.Config{
		ClientID:             cfg.Audience,
		SupportedSigningAlgs: []string{oidc.RS256},
		SkipExpiryCheck:      true,
	})

	return &oidcTokenVerifier{
		verifier:  verifier,
		audience:  cfg.Audience,
		clockSkew: cfg.ClockSkew,
		now:       time.Now,
	}, nil
}

func (v *oidcTokenVerifier) Verify(
	ctx context.Context,
	rawToken string,
) (types.VerifiedIdentity, error) {
	if !isCompactSignedJWT(rawToken) {
		return types.VerifiedIdentity{}, types.ErrInvalidToken
	}

	token, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return types.VerifiedIdentity{}, contextErr
		}
		return types.VerifiedIdentity{}, types.ErrInvalidToken
	}

	var claims accessTokenClaims
	if err := token.Claims(&claims); err != nil {
		return types.VerifiedIdentity{}, types.ErrInvalidToken
	}

	now := v.now()
	if !validTemporalClaims(claims, now, v.clockSkew) {
		return types.VerifiedIdentity{}, types.ErrInvalidToken
	}
	if strings.TrimSpace(token.Subject) == "" {
		return types.VerifiedIdentity{}, types.ErrInvalidToken
	}

	roles, ok := clientRoles(claims.ResourceAccess, v.audience)
	if !ok {
		return types.VerifiedIdentity{}, types.ErrInvalidToken
	}

	return types.VerifiedIdentity{
		Issuer:            token.Issuer,
		Subject:           token.Subject,
		Roles:             roles,
		PreferredUsername: claims.PreferredUsername,
		Name:              claims.Name,
		Email:             claims.Email,
		EmailVerified:     claims.EmailVerified,
	}, nil
}

type accessTokenClaims struct {
	ExpiresAt         json.RawMessage `json:"exp"`
	NotBefore         json.RawMessage `json:"nbf"`
	IssuedAt          json.RawMessage `json:"iat"`
	ResourceAccess    json.RawMessage `json:"resource_access"`
	PreferredUsername string          `json:"preferred_username"`
	Name              string          `json:"name"`
	Email             string          `json:"email"`
	EmailVerified     bool            `json:"email_verified"`
}

func validTemporalClaims(
	claims accessTokenClaims,
	now time.Time,
	clockSkew time.Duration,
) bool {
	expiresAt, ok := numericDate(claims.ExpiresAt)
	if !ok || now.After(expiresAt.Add(clockSkew)) {
		return false
	}

	if len(claims.NotBefore) != 0 {
		notBefore, ok := numericDate(claims.NotBefore)
		if !ok || now.Add(clockSkew).Before(notBefore) {
			return false
		}
	}

	if len(claims.IssuedAt) != 0 {
		issuedAt, ok := numericDate(claims.IssuedAt)
		if !ok || now.Add(clockSkew).Before(issuedAt) {
			return false
		}
	}

	return true
}

func numericDate(raw json.RawMessage) (time.Time, bool) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return time.Time{}, false
	}

	var seconds int64
	if err := json.Unmarshal(raw, &seconds); err != nil {
		return time.Time{}, false
	}
	return time.Unix(seconds, 0), true
}

func clientRoles(resourceAccess json.RawMessage, audience string) ([]string, bool) {
	if !hasJSONShape(resourceAccess, '{', '}') {
		return nil, false
	}

	var resources map[string]json.RawMessage
	if err := json.Unmarshal(resourceAccess, &resources); err != nil {
		return nil, false
	}

	clientAccess, exists := resources[audience]
	if !exists || !hasJSONShape(clientAccess, '{', '}') {
		return nil, false
	}

	var access struct {
		Roles json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(clientAccess, &access); err != nil {
		return nil, false
	}
	if !hasJSONShape(access.Roles, '[', ']') {
		return nil, false
	}

	var roles []string
	if err := json.Unmarshal(access.Roles, &roles); err != nil {
		return nil, false
	}
	return roles, true
}

func hasJSONShape(raw json.RawMessage, first, last byte) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) >= 2 && trimmed[0] == first && trimmed[len(trimmed)-1] == last
}

func isCompactSignedJWT(rawToken string) bool {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
	}
	return true
}

func checkJWKSAvailability(ctx context.Context, client *http.Client, jwksURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return fmt.Errorf("create JWKS request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request JWKS: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request JWKS: unexpected HTTP status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxJWKSResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read JWKS: %w", err)
	}
	if int64(len(body)) > maxJWKSResponseBytes {
		return fmt.Errorf("decode JWKS: response is too large")
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	var document struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}
	if len(document.Keys) == 0 {
		return fmt.Errorf("decode JWKS: no signing keys")
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}
