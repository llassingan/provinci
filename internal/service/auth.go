package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"vps-store/internal/model"
	"vps-store/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAdminAlreadyExists = errors.New("admin already exists")
)

type AuthService struct {
	userRepo      *repository.UserRepository
	encryptionKey []byte
}

func NewAuthService(userRepo *repository.UserRepository, encryptionKeyHex string) (*AuthService, error) {
	key, err := hex.DecodeString(encryptionKeyHex)
	if err != nil {
		return nil, errors.New("invalid encryption key: must be valid hex")
	}
	if len(key) != 32 {
		return nil, errors.New("invalid encryption key: must be 32 bytes")
	}
	return &AuthService{
		userRepo:      userRepo,
		encryptionKey: key,
	}, nil
}

func (s *AuthService) Signup(ctx context.Context, email, password string) (*model.User, string, error) {
	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, "", err
	}
	if count > 0 {
		return nil, "", ErrAdminAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, "", err
	}

	user, err := s.userRepo.CreateUser(ctx, email, string(hash))
	if err != nil {
		return nil, "", err
	}

	token, err := s.createSessionToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.createSessionToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *AuthService) ValidateSession(token string) (int64, error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, ErrInvalidCredentials
	}

	if len(data) < 12 {
		return 0, ErrInvalidCredentials
	}

	nonce := data[:12]
	ciphertext := data[12:]

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return 0, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return 0, err
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return 0, ErrInvalidCredentials
	}

	var payload struct {
		UserID    int64  `json:"user_id"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return 0, ErrInvalidCredentials
	}

	expiresAt, err := time.Parse(time.RFC3339, payload.ExpiresAt)
	if err != nil {
		return 0, ErrInvalidCredentials
	}

	if time.Now().UTC().After(expiresAt) {
		return 0, ErrInvalidCredentials
	}

	return payload.UserID, nil
}

type sessionPayload struct {
	UserID    int64  `json:"user_id"`
	ExpiresAt string `json:"expires_at"`
}

func (s *AuthService) createSessionToken(userID int64) (string, error) {
	payload := sessionPayload{
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	combined := append(nonce, ciphertext...)
	return base64.RawURLEncoding.EncodeToString(combined), nil
}

func init() {
	_ = subtle.ConstantTimeCompare
}
