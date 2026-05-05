package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofer/tui/api"
)

func main() {
	client := api.New("http://localhost:8080")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Register
	username := fmt.Sprintf("t_%d", time.Now().Unix()%1000000)
	user, err := client.Register(ctx, username, "secret123")
	fmt.Printf("Register(%q): user=%+v err=%v\n", username, user, err)

	// 2. Register again — должна быть ошибка ErrConflict
	_, err = client.Register(ctx, username, "secret123")
	fmt.Printf("Register again: err=%v; is AlreadyExists=%v\n",
		err, errors.Is(err, api.ErrConflict))

	// 3. Login — успех.
	loginResp, err := client.Login(ctx, username, "secret123")
	fmt.Printf("Login: user=%+v access=%.20s... err=%v\n",
		loginResp.User, loginResp.Tokens.AccessToken, err)

	// 4. Login с неверным паролем — ErrInvalidCredentials.
	_, err = client.Login(ctx, username, "wrong")
	fmt.Printf("Login wrong: err=%v; is InvalidCreds=%v\n",
		err, errors.Is(err, api.ErrInvalidCredentials))

	// 5. Login несуществующего — ErrNotFound.
	_, err = client.Login(ctx, "nobody_xyz", "whatever")
	fmt.Printf("Login nobody: err=%v; is NotFound=%v\n",
		err, errors.Is(err, api.ErrNotFound))

	// 6. ListChannels без авторизации → 401/ErrInvalidCredentials.
	_, err = client.ListChannels(ctx)
	fmt.Printf("ListChannels (no auth): err=%v\n", err)

	// 7. Логинимся и устанавливаем токен в клиент.
	loginResp, err = client.Login(ctx, username, "secret123")
	if err != nil {
		fmt.Printf("pre-login failed: %v\n", err)
		return
	}
	client.SetAuth(loginResp.Tokens.AccessToken)

	// 8. ListChannels с токеном — должно работать.
	channels, err := client.ListChannels(ctx)
	fmt.Printf("ListChannels (authed): count=%d err=%v\n", len(channels), err)
	for _, ch := range channels {
		fmt.Printf("  - %s (id=%s, by=%s)\n", ch.Name, ch.ID, ch.CreatedBy)
	}
}
