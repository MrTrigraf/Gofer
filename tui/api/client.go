// Package api — HTTP-клиент для общения TUI с gofer-сервером.
//
// Предоставляет типизированные методы для REST-эндпоинтов.
// Ошибки сервера (401, 404, 409) транслируются в sentinel-errors
// (см. errors.go), которые UI может сравнивать через errors.Is().
//
// Токен авторизации хранится внутри клиента (см. SetAuth) и
// автоматически подкладывается в заголовок Authorization.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Client — HTTP-клиент к gofer-серверу.
//
// Безопасен для использования из нескольких горутин (http.Client thread-safe).
// Токен под mutex'ом, потому что SetAuth вызывается из Update,
// а чтение — из горутин, выполняющих запросы.
type Client struct {
	baseURL    string
	httpClient *http.Client

	mu          sync.RWMutex
	accessToken string
}

// New — создаёт клиента, указывающего на baseURL.
//
// Пример: api.New("http://localhost:8080")
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// SetAuth — устанавливает access-токен, который будет отправляться
// в заголовке Authorization для всех последующих запросов.
//
// Пустая строка сбрасывает авторизацию.
func (c *Client) SetAuth(accessToken string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = accessToken
}

// authHeader — читает текущий токен под RLock.
func (c *Client) authHeader() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}

// errorResponse — формат JSON-ошибок сервера (pkg/httputil.ErrorResponse).
type errorResponse struct {
	Error string `json:"error"`
}

// do — общая реализация HTTP-запроса:
//  1. сериализует body в JSON (если есть);
//  2. подкладывает токен в заголовок Authorization (если установлен);
//  3. выполняет запрос с контекстом;
//  4. транслирует HTTP-код в sentinel-ошибку;
//  5. парсит JSON-ответ в out (если передан).
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reqBody *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("api.do: marshal body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("api.do: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Автоматическая авторизация: токен есть → Bearer.
	if token := c.authHeader(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("api.do: %w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return translateError(resp)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("api.do: decode response: %w", err)
		}
	}
	return nil
}

// translateError — превращает HTTP-ответ сервера в sentinel-ошибку.
//
// UI различает "неверный пароль" (красная строка) / "user не найден"
// (другое сообщение) / "сервер упал" (третье) — через errors.Is.
func translateError(resp *http.Response) error {
	var msg string
	var errResp errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
		msg = errResp.Error
	}

	switch resp.StatusCode {
	case http.StatusBadRequest: // 400
		return fmt.Errorf("%w: %s", ErrBadRequest, msg)
	case http.StatusUnauthorized: // 401
		return fmt.Errorf("%w: %s", ErrInvalidCredentials, msg)
	case http.StatusForbidden: // 403
		return fmt.Errorf("%w: %s", ErrForbidden, msg)
	case http.StatusNotFound: // 404
		return fmt.Errorf("%w: %s", ErrNotFound, msg)
	case http.StatusConflict: // 409
		return fmt.Errorf("%w: %s", ErrConflict, msg)
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: status=%d msg=%q", ErrServer, resp.StatusCode, msg)
	}

	return fmt.Errorf("%w: status=%d msg=%q", ErrUnexpected, resp.StatusCode, msg)
}
