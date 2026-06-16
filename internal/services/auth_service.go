package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/embrionix/dashboard/internal/config"
	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/repositories"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

// AuthService handles password hashing and JWT issue/verify.
type AuthService struct {
	users *repositories.UserRepository
	cfg   config.AuthConfig
}

func NewAuthService(users *repositories.UserRepository, cfg config.AuthConfig) *AuthService {
	return &AuthService{users: users, cfg: cfg}
}

func (s *AuthService) Enabled() bool { return s.cfg.Enabled }

// HashPassword returns a bcrypt hash for the given plaintext password.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

// Authenticate verifies credentials and returns a signed JWT plus the user.
func (s *AuthService) Authenticate(username, password string) (string, *models.User, error) {
	user, err := s.users.FindByUsername(username)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", nil, ErrInvalidCredentials
	}
	token, err := s.issue(user)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

type claims struct {
	UserID   uint        `json:"uid"`
	Username string      `json:"username"`
	Role     models.Role `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthService) issue(user *models.User) (string, error) {
	ttl := time.Duration(s.cfg.TokenTTLHours) * time.Hour
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	c := claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(s.cfg.JWTSecret))
}

// Verify parses a token and returns its username and role.
func (s *AuthService) Verify(token string) (username string, role models.Role, err error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return "", "", err
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return "", "", errors.New("invalid token")
	}
	return c.Username, c.Role, nil
}

// APIKeyMatches reports whether key matches the configured static API key.
func (s *AuthService) APIKeyMatches(key string) bool {
	return s.cfg.APIKey != "" && key == s.cfg.APIKey
}
