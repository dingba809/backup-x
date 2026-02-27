package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	sessionCookieName = "backup_x_session"
	sessionMaxAge     = 24 * time.Hour
)

type sessionData struct {
	Username  string
	Role      string
	ExpiresAt time.Time
}

var (
	sessions    = make(map[string]*sessionData)
	sessionLock sync.RWMutex
)

// generateToken 生成安全随机 token
func generateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSession 创建 session 并设置 Cookie
func CreateSession(w http.ResponseWriter, username, role string) error {
	token, err := generateToken()
	if err != nil {
		return err
	}

	sessionLock.Lock()
	sessions[token] = &sessionData{
		Username:  username,
		Role:      role,
		ExpiresAt: time.Now().Add(sessionMaxAge),
	}
	sessionLock.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(sessionMaxAge.Seconds()),
	})

	return nil
}

// GetSession 从 Cookie 获取 session
func GetSession(r *http.Request) *sessionData {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}

	sessionLock.RLock()
	sd, ok := sessions[cookie.Value]
	sessionLock.RUnlock()

	if !ok {
		return nil
	}

	// 检查过期
	if time.Now().After(sd.ExpiresAt) {
		sessionLock.Lock()
		delete(sessions, cookie.Value)
		sessionLock.Unlock()
		return nil
	}

	return sd
}

// DestroySession 删除 session（登出）
func DestroySession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return
	}

	sessionLock.Lock()
	delete(sessions, cookie.Value)
	sessionLock.Unlock()

	// 清除 Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// SessionAuth Session 认证中间件
func SessionAuth(f ViewFunc) ViewFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sd := GetSession(r)
		if sd == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		f(w, r)
	}
}

// AdminOnly 仅 admin 角色可访问的中间件
func AdminOnly(f ViewFunc) ViewFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sd := GetSession(r)
		if sd == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if sd.Role != "admin" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("权限不足，仅管理员可访问"))
			return
		}
		f(w, r)
	}
}

// CleanExpiredSessions 清理过期 session（可定期调用）
func CleanExpiredSessions() {
	sessionLock.Lock()
	defer sessionLock.Unlock()

	now := time.Now()
	for token, sd := range sessions {
		if now.After(sd.ExpiresAt) {
			delete(sessions, token)
		}
	}
}
