package screen

import tea "github.com/charmbracelet/bubbletea"

// Hitbox — кликабельная зона на экране.
//
// Координаты — в ЯЧЕЙКАХ терминала (не пикселях), включительно с обеих сторон.
// (X1, Y1) — левый верхний угол, (X2, Y2) — правый нижний.
//
// ID — строковый идентификатор того, что произойдёт при клике
// (например, "close", "btn_login", "tab_channels").
type Hitbox struct {
	X1, Y1 int
	X2, Y2 int
	ID     string
}

// Contains — проверяет, попадает ли точка (x, y) внутрь хитбокса.
func (h Hitbox) Contains(x, y int) bool {
	return x >= h.X1 && x <= h.X2 && y >= h.Y1 && y <= h.Y2
}

// HitTest — пробегает все хитбоксы и возвращает ID того,
// в который попал клик. Если ни в один не попал — пустая строка.
//
// Перебираем В ОБРАТНОМ порядке: если хитбоксы перекрываются,
// побеждает тот, что был добавлен ПОЗЖЕ — это обычно "тот, что
// нарисован поверх" (например, кнопка попапа поверх фона).
func HitTest(boxes []Hitbox, x, y int) string {
	for i := len(boxes) - 1; i >= 0; i-- {
		if boxes[i].Contains(x, y) {
			return boxes[i].ID
		}
	}
	return ""
}

// Screen — общий интерфейс для всех экранов приложения
// (login, home, chat, ...).
//
// Главная Model в пакете tui хранит ОДНУ переменную типа Screen
// и делегирует ей события и рендер. Когда нужно сменить экран —
// просто подменяет переменную.
//
// Контракт вызовов:
//  1. SetSize(w, h) и SetOrigin(x, y) вызываются ПЕРЕД View.
//     Экран обязан перерисовываться под новые размеры
//     и регистрировать хитбоксы в координатах ВСЕГО терминала.
//  2. View() возвращает строку — содержимое тела окна.
//  3. Hitboxes() возвращает зарегистрированные при последнем View
//     кликабельные зоны.
type Screen interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Screen, tea.Cmd)
	View() string
	Hitboxes() []Hitbox
	SetSize(width, height int)
	SetOrigin(x, y int)
}
