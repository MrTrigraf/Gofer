// Package auth — состояние авторизованного пользователя
// и сообщение, сигнализирующее о успешной аутентификации.
//
// Отдельный пакет, чтобы и views (кто создаёт сообщение),
// и tui (кто его ловит и меняет экраны) могли импортировать,
// не образуя цикла.
package auth

// AuthState — то, что клиент помнит о текущем юзере.
//
// Хранится в главной модели; передаётся экранам, которым
// нужны токен (для HTTP) или данные юзера (для отображения).
type AuthState struct {
	UserID       string
	Username     string
	AccessToken  string
	RefreshToken string
}

// IsAuthenticated — удобный helper для проверки "есть ли сессия".
// UserID пустой значит юзер не вошёл.
func (a AuthState) IsAuthenticated() bool {
	return a.UserID != ""
}

// AuthenticatedMsg — отправляется LoginModel'ом, когда сервер
// подтвердил логин или регистрацию-с-автологином. Ловится главной
// моделью: она сохраняет AuthState и переключает экран.
type AuthenticatedMsg struct {
	State AuthState
}
