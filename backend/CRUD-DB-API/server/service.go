package server

import (
	"context"
	"crypto/subtle"
	"time"
)

// ---------------------------------------------------------------------------
// AuthService — interface for all authentication operations (ISP / DIP)
// ---------------------------------------------------------------------------

// AuthService abstracts authentication, token management, and 2FA operations.
type AuthService interface {
	CheckAdmin(password, email string) (bool, string)
	CheckJury(password, email string) (bool, string)
	GenerateAndStoreToken() (string, error)
	VerifyToken(token string) bool
	Generate2FAAndSend(ctx context.Context, email, role string) error
	Verify2FA(email, code string) bool
	SendVerificationMail(email, token string) error
}

// ---------------------------------------------------------------------------
// authService — implementation backed by package-level state in auth.go
// ---------------------------------------------------------------------------

// authService implements AuthService. It delegates to the package-level state
// variables (storedTokens, pendingVerifications) so that state persists across
// the lifetime of the process regardless of how many Handlers instances exist.
type authService struct{}

func (s *authService) CheckAdmin(password, email string) (bool, string) {
	return CheckAccessAdmin(password, email)
}

func (s *authService) CheckJury(password, email string) (bool, string) {
	return CheckAccessJury(password, email)
}

func (s *authService) GenerateAndStoreToken() (string, error) {
	return generateAndStoreToken()
}

func (s *authService) VerifyToken(token string) bool {
	tokenMu.Lock()
	_, exists := storedTokens[token]
	if exists {
		evictToken(token)
	}
	tokenMu.Unlock()
	return exists
}

func (s *authService) Generate2FAAndSend(ctx context.Context, email, role string) error {
	code, err := generate2FACode()
	if err != nil {
		return err
	}
	verifyMu.Lock()
	pendingVerifications[email] = &PendingVerification{
		Code:      code,
		Role:      role,
		CreatedAt: time.Now(),
	}
	verifyMu.Unlock()
	return requestVerificationMail(email, code)
}

func (s *authService) Verify2FA(email, code string) bool {
	verifyMu.Lock()
	defer verifyMu.Unlock()

	pending, exists := pendingVerifications[email]
	if !exists {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(pending.Code), []byte(code)) != 1 {
		return false
	}
	if time.Since(pending.CreatedAt) > 5*time.Minute {
		return false
	}

	delete(pendingVerifications, email)
	for k, v := range pendingVerifications {
		if time.Since(v.CreatedAt) > 5*time.Minute {
			delete(pendingVerifications, k)
		}
	}
	return true
}

func (s *authService) SendVerificationMail(email, token string) error {
	return requestVerificationMail(email, token)
}
