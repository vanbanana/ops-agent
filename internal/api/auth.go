package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

// AuthConfig holds JWT authentication settings.
type AuthConfig struct {
	Secret   string
	Username string
	Password string
	TokenTTL time.Duration
}

// AuthService handles login and token validation.
type AuthService struct {
	cfg          AuthConfig
	failedLogins map[string]*loginAttempts // IP → attempts
	mu           sync.Mutex
}

type loginAttempts struct {
	count    int
	lockedAt time.Time
}

const maxLoginAttempts = 5
const lockDuration = 3 * time.Minute

// NewAuthService creates an auth service.
func NewAuthService(cfg AuthConfig) *AuthService {
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = 24 * time.Hour
	}
	return &AuthService{
		cfg:          cfg,
		failedLogins: make(map[string]*loginAttempts),
	}
}

// HandleLogin processes POST /api/v1/auth/login.
func (a *AuthService) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]any{"code": 400, "error": "invalid body", "error_code": "AUTH_LOGIN_001"})
		return
	}

	ip := extractIP(r)

	// Check lock
	a.mu.Lock()
	attempts := a.failedLogins[ip]
	if attempts != nil && attempts.count >= maxLoginAttempts {
		if time.Since(attempts.lockedAt) < lockDuration {
			remaining := lockDuration - time.Since(attempts.lockedAt)
			a.mu.Unlock()
			writeJSON(w, 429, map[string]any{
				"code": 429, "error": "IP已锁定", "error_code": "AUTH_RATE_001",
				"remaining_seconds": int(remaining.Seconds()),
			})
			return
		}
		// Lock expired, reset
		attempts.count = 0
	}
	a.mu.Unlock()

	// Validate credentials
	if req.Username != a.cfg.Username || req.Password != a.cfg.Password {
		a.mu.Lock()
		if a.failedLogins[ip] == nil {
			a.failedLogins[ip] = &loginAttempts{}
		}
		a.failedLogins[ip].count++
		if a.failedLogins[ip].count >= maxLoginAttempts {
			a.failedLogins[ip].lockedAt = time.Now()
		}
		a.mu.Unlock()

		writeJSON(w, 401, map[string]any{"code": 401, "error": "用户名或密码错误", "error_code": "AUTH_LOGIN_001"})
		return
	}

	// Reset failed attempts on success
	a.mu.Lock()
	delete(a.failedLogins, ip)
	a.mu.Unlock()

	// Generate JWT
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  req.Username,
		"role": "admin",
		"iat":  now.Unix(),
		"exp":  now.Add(a.cfg.TokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(a.cfg.Secret))
	if err != nil {
		writeJSON(w, 500, map[string]any{"code": 500, "error": "token generation failed", "error_code": "API_INTERNAL_001"})
		return
	}

	writeJSON(w, 200, map[string]any{
		"code": 0,
		"data": map[string]any{
			"token":      tokenStr,
			"expires_at": now.Add(a.cfg.TokenTTL).Format(time.RFC3339),
			"user":       map[string]any{"username": req.Username, "role": "admin"},
		},
	})
}

// JWTMiddleware validates Bearer token on protected routes.
func (a *AuthService) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJSON(w, 401, map[string]any{"code": 401, "error": "未登录", "error_code": "AUTH_TOKEN_001"})
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			return []byte(a.cfg.Secret), nil
		})
		if err != nil || !token.Valid {
			writeJSON(w, 401, map[string]any{"code": 401, "error": "登录已过期", "error_code": "AUTH_TOKEN_001"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func extractIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

// HandleUnlockIP handles DELETE /api/v1/auth/lockout/{ip} -- admin unlocks an IP.
// Only accessible from localhost for security.
func (a *AuthService) HandleUnlockIP(w http.ResponseWriter, r *http.Request) {
	ip := chi.URLParam(r, "ip")
	requesterIP := extractIP(r)
	if requesterIP != "127.0.0.1" && requesterIP != "::1" && requesterIP != "localhost" {
		writeJSON(w, 403, map[string]any{"code": 403, "error": "only localhost can unlock"})
		return
	}
	a.mu.Lock()
	delete(a.failedLogins, ip)
	a.mu.Unlock()
	writeJSON(w, 200, map[string]any{"code": 0, "message": "IP unlocked"})
}

// HandleListLockouts handles GET /api/v1/auth/lockouts -- admin lists locked IPs.
func (a *AuthService) HandleListLockouts(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	type lockoutInfo struct {
		IP        string `json:"ip"`
		Attempts  int    `json:"attempts"`
		LockedAt  string `json:"locked_at,omitempty"`
		Remaining int    `json:"remaining_seconds,omitempty"`
	}
	var result []lockoutInfo
	for ip, att := range a.failedLogins {
		info := lockoutInfo{IP: ip, Attempts: att.count}
		if att.count >= maxLoginAttempts {
			info.LockedAt = att.lockedAt.Format(time.RFC3339)
			remaining := lockDuration - time.Since(att.lockedAt)
			if remaining > 0 {
				info.Remaining = int(remaining.Seconds())
			}
		}
		result = append(result, info)
	}
	if result == nil {
		result = []lockoutInfo{}
	}
	writeJSON(w, 200, map[string]any{"code": 0, "data": result})
}
