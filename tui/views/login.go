package views

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/styles"
)

// === СОБСТВЕННЫЕ СООБЩЕНИЯ ЭКРАНА ===
//
// Эти типы прилетают в Update() как результат асинхронных команд.
// Четыре отдельных типа лучше одного "общего" — компилятор проверит,
// что мы не перепутали поля в обработчиках.
type LoginErrorMsg struct {
	Err error
}

type RegisterSuccessMsg struct {
	User api.User
}

type RegisterErrorMsg struct {
	Err error
}

// === focus ===

type focusItem int

const (
	focusUsername focusItem = iota
	focusPassword
	focusLogin
	focusRegister
	focusItemCount
)

// === РАЗМЕРЫ ===

const (
	contentWidth  = 40
	contentHeight = 15
)

// === МОДЕЛЬ ===

type LoginModel struct {
	usernameInput textinput.Model
	passwordInput textinput.Model
	focused       focusItem

	// HTTP-клиент, инжектится снаружи.
	apiClient *api.Client

	// errorLine — красное сообщение под формой.
	// Пока используется и для успехов (зелёным), и для ошибок (красным).
	// Отдельного поля для попапа пока нет — сделаем в Заходе 3.
	errorLine string
	errorOK   bool // если true — errorLine показывается зелёным (успех)

	// loading — true, пока идёт HTTP-запрос.
	// Во время loading кнопки не срабатывают, под формой показан индикатор.
	loading bool

	width, height    int
	originX, originY int

	hitboxes []screen.Hitbox
}

// NewLogin принимает api-клиент (DI).
func NewLogin(apiClient *api.Client) *LoginModel {
	usernameInput := textinput.New()
	usernameInput.Placeholder = "username"
	usernameInput.Prompt = ""
	usernameInput.CharLimit = 16
	usernameInput.Width = 21 //24

	passwordInput := textinput.New()
	passwordInput.Placeholder = "password"
	passwordInput.Prompt = ""
	passwordInput.CharLimit = 64
	passwordInput.Width = 21 //24
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '•'

	m := &LoginModel{
		usernameInput: usernameInput,
		passwordInput: passwordInput,
		focused:       focusUsername,
		apiClient:     apiClient,
	}
	m.applyFocus()
	return m
}

// === Реализация интерфейса screen.Screen ===

func (m *LoginModel) Init() tea.Cmd             { return nil }
func (m *LoginModel) SetSize(width, height int) { m.width, m.height = width, height }
func (m *LoginModel) SetOrigin(x, y int)        { m.originX, m.originY = x, y }
func (m *LoginModel) Hitboxes() []screen.Hitbox { return m.hitboxes }

// === ФОКУС ===

func (m *LoginModel) applyFocus() {
	if m.focused == focusUsername {
		m.usernameInput.Focus()
	} else {
		m.usernameInput.Blur()
	}
	if m.focused == focusPassword {
		m.passwordInput.Focus()
	} else {
		m.passwordInput.Blur()
	}
}

func (m *LoginModel) nextFocus() {
	m.focused = (m.focused + 1) % focusItemCount
	m.applyFocus()
}

func (m *LoginModel) prevFocus() {
	m.focused = (m.focused - 1 + focusItemCount) % focusItemCount
	m.applyFocus()
}

func (m *LoginModel) focusedID() string {
	switch m.focused {
	case focusUsername:
		return "input_username"
	case focusPassword:
		return "input_password"
	case focusLogin:
		return "btn_login"
	case focusRegister:
		return "btn_register"
	}
	return ""
}

// === АКТИВАЦИЯ (Enter / клик) ===

func (m *LoginModel) activate(id string) tea.Cmd {
	switch id {
	case "btn_login":
		if m.loading {
			return nil
		}
		// Лёгкая клиентская валидация перед HTTP —
		// чтобы не гонять сервер на очевидно пустых данных.
		if m.usernameInput.Value() == "" || m.passwordInput.Value() == "" {
			m.errorLine = "username and password required"
			m.errorOK = false
			return nil
		}
		m.loading = true
		m.errorLine = ""
		return loginCmd(m.apiClient, m.usernameInput.Value(), m.passwordInput.Value())

	case "btn_register":
		if m.loading {
			return nil
		}
		if m.usernameInput.Value() == "" || m.passwordInput.Value() == "" {
			m.errorLine = "username and password required"
			m.errorOK = false
			return nil
		}
		m.loading = true
		m.errorLine = ""
		return registerCmd(m.apiClient, m.usernameInput.Value(), m.passwordInput.Value())

	case "input_username":
		m.focused = focusUsername
		m.applyFocus()
	case "input_password":
		m.focused = focusPassword
		m.applyFocus()
	}
	return nil
}

// === КОМАНДЫ (асинхронные HTTP-запросы) ===

// loginCmd возвращает tea.Cmd, которая в горутине сделает запрос
// и вернёт либо LoginSuccessMsg, либо LoginErrorMsg.
//
// Bubble Tea сам запускает эту функцию в горутине и перехватывает Msg.
func loginCmd(client *api.Client, username, password string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := client.Login(ctx, username, password)
		if err != nil {
			return LoginErrorMsg{Err: err}
		}
		return auth.AuthenticatedMsg{
			State: auth.AuthState{
				UserID:       resp.User.ID,
				Username:     resp.User.Username,
				AccessToken:  resp.Tokens.AccessToken,
				RefreshToken: resp.Tokens.RefreshToken,
			},
		}
	}
}

func registerCmd(client *api.Client, username, password string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		user, err := client.Register(ctx, username, password)
		if err != nil {
			return RegisterErrorMsg{Err: err}
		}
		return RegisterSuccessMsg{User: user}
	}
}

// === UPDATE ===

func (m *LoginModel) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case LoginErrorMsg:
		m.loading = false
		m.errorLine = mapLoginError(msg.Err)
		m.errorOK = false
		return m, nil
	case RegisterSuccessMsg:
		m.loading = false
		m.errorLine = fmt.Sprintf("✓ registered as %q — you can log in now", msg.User.Username)
		m.errorOK = true
		// Для удобства: очистим пароль, чтобы юзер ввёл новый.
		m.passwordInput.SetValue("")
		// И переведём фокус на кнопку LOGIN — следующий естественный шаг.
		m.focused = focusLogin
		m.applyFocus()
		return m, nil
	case RegisterErrorMsg:
		m.loading = false
		m.errorLine = mapRegisterError(msg.Err)
		m.errorOK = false
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.nextFocus()
			return m, nil
		case "shift+tab", "up":
			m.prevFocus()
			return m, nil
		case "enter":
			id := m.focusedID()
			if id == "input_username" || id == "input_password" {
				id = "btn_login"
			}
			return m, m.activate(id)
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		if id := screen.HitTest(m.hitboxes, msg.X, msg.Y); id != "" {
			return m, m.activate(id)
		}
		return m, nil
	}

	// Прочие сообщения — в активный textinput.
	// Во время loading тоже пропускаем, чтобы юзер не мог стереть свои данные
	// посреди запроса (субъективное решение, можно убрать).
	if m.loading {
		return m, nil
	}

	var cmd tea.Cmd
	switch m.focused {
	case focusUsername:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	case focusPassword:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	return m, cmd
}

// mapLoginError — превращает api-error в человеческое сообщение.
func mapLoginError(err error) string {
	switch {
	case errors.Is(err, api.ErrInvalidCredentials),
		errors.Is(err, api.ErrNotFound):
		// По UX обычно не раскрывают, что именно не так —
		// объединяем оба случая в одно сообщение.
		return "⚠ Invalid username or password"
	case errors.Is(err, api.ErrUnreachable):
		return "⚠ Server unreachable"
	case errors.Is(err, api.ErrBadRequest):
		// Текст приходит от сервера ("username must be 1..16 characters")
		return "⚠ " + err.Error()
	case errors.Is(err, api.ErrServer):
		return "⚠ Server error, try again later"
	}
	return "⚠ " + err.Error()
}

func mapRegisterError(err error) string {
	switch {
	case errors.Is(err, api.ErrConflict):
		return "⚠ Username already taken"
	case errors.Is(err, api.ErrUnreachable):
		return "⚠ Server unreachable"
	case errors.Is(err, api.ErrBadRequest):
		return "⚠ " + err.Error()
	case errors.Is(err, api.ErrServer):
		return "⚠ Server error, try again later"
	}
	return "⚠ " + err.Error()
}

// === VIEW ===

func (m *LoginModel) View() string {
	title := styles.StyleTitle.Render("◈ LOGIN / REGISTER")

	usernameField := m.renderField("Username:", m.usernameInput.View(), m.focused == focusUsername)
	passwordField := m.renderField("Password:", m.passwordInput.View(), m.focused == focusPassword)
	form := lipgloss.JoinVertical(lipgloss.Left, usernameField, passwordField)

	loginBtn := m.renderButton("[ LOGIN ]", m.focused == focusLogin)
	registerBtn := m.renderButton("[ REGISTER ]", m.focused == focusRegister)
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, loginBtn, "    ", registerBtn)

	// Статусная строка под формой.
	//   loading=true          → "◎ connecting..."
	//   errorLine + OK        → зелёным
	//   errorLine + не OK     → красным
	//   иначе                 → пробел (резерв места, чтобы не "прыгало")
	status := " "
	switch {
	case m.loading:
		status = styles.StyleAccent.Render("◎ connecting...")
	case m.errorLine != "" && m.errorOK:
		status = styles.StyleOK.Render(m.errorLine)
	case m.errorLine != "":
		status = styles.StyleDanger.Render(m.errorLine)
	}

	block := func(s string) string {
		return lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, s)
	}

	rows := []string{
		block(title),
		"",
		block(form),
		"",
		block(buttons),
		"",
		block(status),
	}
	inner := lipgloss.JoinVertical(lipgloss.Left, rows...)

	content := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Render(inner)

	placed := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)

	m.registerHitboxes(usernameField, passwordField, loginBtn, registerBtn,
		title, form, buttons)

	return placed
}

func (m *LoginModel) renderField(label, input string, focused bool) string {
	style := styles.StyleInput
	if focused {
		style = styles.StyleInputFocused
	}

	// Фиксируем ширину САМОГО содержимого поля (24 ячейки = textinput.Width).
	// Оборачиваем input в блок фикс-ширины ДО применения рамки —
	// это гарантирует, что рамка обнимет ровно 24 ячейки независимо
	// от состояния textinput (пустое / с текстом / в фокусе).
	fixedInput := lipgloss.NewStyle().Width(24).Render(input)
	bordered := style.Render(fixedInput)

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		styles.StyleFaint.Render(label+" "),
		bordered,
	)
}

func (m *LoginModel) renderButton(label string, focused bool) string {
	if focused {
		return styles.StyleButtonFocused.Render(label)
	}
	return styles.StyleButton.Render(label)
}

func (m *LoginModel) registerHitboxes(
	usernameField, passwordField, loginBtn, registerBtn string,
	title, form, buttons string,
) {
	m.hitboxes = m.hitboxes[:0]

	contentX := m.originX + (m.width-contentWidth)/2
	contentY := m.originY + (m.height-contentHeight)/2

	titleH := lipgloss.Height(title)
	formH := lipgloss.Height(form)
	usernameH := lipgloss.Height(usernameField)

	formY := contentY + titleH + 1
	usernameY := formY
	passwordY := formY + usernameH

	usernameW := lipgloss.Width(usernameField)
	passwordW := lipgloss.Width(passwordField)

	usernameX := contentX + (contentWidth-usernameW)/2
	passwordX := contentX + (contentWidth-passwordW)/2

	usernameBoxH := lipgloss.Height(usernameField)
	passwordBoxH := lipgloss.Height(passwordField)

	m.hitboxes = append(m.hitboxes,
		screen.Hitbox{
			X1: usernameX, Y1: usernameY,
			X2: usernameX + usernameW - 1, Y2: usernameY + usernameBoxH - 1,
			ID: "input_username",
		},
		screen.Hitbox{
			X1: passwordX, Y1: passwordY,
			X2: passwordX + passwordW - 1, Y2: passwordY + passwordBoxH - 1,
			ID: "input_password",
		},
	)

	buttonsY := formY + formH + 1
	buttonsW := lipgloss.Width(buttons)
	buttonsX := contentX + (contentWidth-buttonsW)/2

	loginW := lipgloss.Width(loginBtn)
	registerW := lipgloss.Width(registerBtn)
	loginH := lipgloss.Height(loginBtn)

	loginX := buttonsX
	registerX := buttonsX + loginW + 4

	m.hitboxes = append(m.hitboxes,
		screen.Hitbox{
			X1: loginX, Y1: buttonsY,
			X2: loginX + loginW - 1, Y2: buttonsY + loginH - 1,
			ID: "btn_login",
		},
		screen.Hitbox{
			X1: registerX, Y1: buttonsY,
			X2: registerX + registerW - 1, Y2: buttonsY + loginH - 1,
			ID: "btn_register",
		},
	)
}
