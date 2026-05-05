package api

import (
	"context"
	"net/http"
	"time"
)

// TokenPair — ответ сервера на login/refresh.
//
// Структура совпадает с server-side pkg/jwt.TokenPair,
// но держим СВОЮ копию, чтобы клиент не зависел от internal
// сервера (они общаются только через стабильный JSON-контракт).
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// User — данные пользователя (ответ на register, часть LoginResponse).
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginResponse — ответ сервера на /api/v1/auth/login.
// Содержит и данные юзера (для UI), и токены (для последующих запросов).
type LoginResponse struct {
	User   User      `json:"user"`
	Tokens TokenPair `json:"tokens"`
}

// Login — POST /api/v1/auth/login.
// Возвращает данные юзера + пару токенов при успехе.
func (c *Client) Login(ctx context.Context, username, password string) (LoginResponse, error) {
	body := map[string]string{
		"username": username,
		"password": password,
	}

	var resp LoginResponse
	err := c.do(ctx, http.MethodPost, "/api/v1/auth/login", body, &resp)
	return resp, err
}

// Register — POST /api/v1/auth/register.
//
// Возвращает User при успехе или:
//   - ErrConflict          — username занят (409)
//   - ErrBadRequest        — валидация не прошла (400)
//   - ErrUnreachable       — сервер недоступен
//   - ErrServer            — 5xx
func (c *Client) Register(ctx context.Context, username, password string) (User, error) {
	body := map[string]string{
		"username": username,
		"password": password,
	}

	var resp User
	err := c.do(ctx, http.MethodPost, "/api/v1/auth/register", body, &resp)
	return resp, err
}

// Health — GET /api/v1/health.
//
// Проверка доступности сервера. Используется периодическим пингом
// из UI, чтобы показывать NETLINK-индикатор.
// Возвращает nil при 200 OK, иначе ErrUnreachable или ErrServer.
func (c *Client) Health(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/api/v1/health", nil, nil)
}
