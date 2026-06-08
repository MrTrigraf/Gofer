package styles

import "github.com/charmbracelet/lipgloss"

const SidebarWidth = 22

// cyberpunk2077
// const (
// 	ColorBackground lipgloss.Color = "#0A0E14" // основной тёмный фон
// 	ColorText       lipgloss.Color = "#E5E5E5" // основной цвет текста
// 	ColorPrimary    lipgloss.Color = "#FCEE0A" // главный акцент — рамка, заголовки, логотип
// 	ColorSecondary  lipgloss.Color = "#00F0FF" // второстепенный акцент — поля ввода, табы, ссылки
// 	ColorAccent     lipgloss.Color = "#FF2A6D" // подсветка — выделенные элементы, ники
// 	ColorDanger     lipgloss.Color = "#FF003C" // ошибки и опасные действия
// 	ColorSuccess    lipgloss.Color = "#00FF9F" // успешные операции
// 	ColorMuted      lipgloss.Color = "#4A5568" // приглушённый — неактивные элементы, плейсхолдеры
// )

// Kanagawa Wave
const (
	ColorBackground lipgloss.Color = "#1F1F28" // sumiInk1 — основной тёмный фон
	ColorText       lipgloss.Color = "#DCD7BA" // fujiWhite — основной цвет текста
	ColorPrimary    lipgloss.Color = "#C0A36E" // carpYellow — главный акцент
	ColorSecondary  lipgloss.Color = "#7FB4CA" // springBlue — второстепенный акцент
	ColorAccent     lipgloss.Color = "#D27E99" // sakuraPink — подсветка
	ColorDanger     lipgloss.Color = "#E82424" // samuraiRed — ошибки и опасные действия
	ColorSuccess    lipgloss.Color = "#98BB6C" // springGreen — успешные операции
	ColorMuted      lipgloss.Color = "#727169" // fujiGray — приглушённый
)

// === ТЕКСТОВЫЕ СТИЛИ ===
var (
	// заголовки и логотип "◈ GOFER".
	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// акцентный текст (cyan, для ссылок и подсказок).
	StyleAccent = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// приглушённый текст (плейсхолдеры, "click to copy").
	StyleFaint = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// текст ошибок ("⚠ Invalid username...").
	StyleDanger = lipgloss.NewStyle().
			Foreground(ColorDanger)

	// текст успешных операций ("✓ Registration successful").
	StyleOK = lipgloss.NewStyle().
		Foreground(ColorSuccess)

	// отображение ника пользователя в чате.
	StyleUsername = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)
)

// === КАРКАС ПРИЛОЖЕНИЯ ===
var (
	// внешняя двойная рамка ╔═╗║╚╝, охватывающая весь экран.
	StyleAppFrame = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary)

	// верхняя панель с логотипом и [✕].
	// это разделитель ╠═══╣ внутри рамки.
	StyleHeader = lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	// нижняя панель со статусом пользователя / подсказкой.
	// второй ╠═══╣ разделитель.
	StyleFooter = lipgloss.NewStyle().
			BorderTop(true).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	// основное тело между Header и Footer.
	StyleBody = lipgloss.NewStyle().
			Padding(1, 2)
)

// === КНОПКИ ===
var (
	// обычная кнопка [ LOGIN ].
	StyleButton = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Foreground(ColorText).
			Padding(0, 2)

	// кнопка под фокусом (Tab) или наведением мыши.
	StyleButtonFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Foreground(ColorPrimary).
				Bold(true).
				Padding(0, 2)

	// крестик [✕] в углу окна.
	StyleButtonClose = lipgloss.NewStyle().
				Foreground(ColorDanger).
				Bold(true)

	// кнопка [↑] для отправки сообщения.
	StyleButtonSend = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	// индикатор активного соединения [● NETLINK ON].
	StyleButtonNetOn = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	// индикатор разорванного соединения.
	StyleButtonNetOff = lipgloss.NewStyle().
				Foreground(ColorDanger).
				Bold(true)
)

// === ПОЛЯ ВВОДА ===
var (
	// обычное поле ввода без фокуса.
	StyleInput = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1)

	// поле ввода под фокусом (cyan-рамка).
	StyleInputFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSecondary).
				Padding(0, 1)
)

// === САЙДБАР (для экранов с каналами/DM) ===
var (
	// левая панель списка каналов.
	StyleSidebar = lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorMuted)

	// заголовки секций в сайдбаре.
	StyleSectionHeader = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	// выбранный элемент в списке.
	StyleItemActive = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	// невыбранные элементы.
	StyleItemInactive = lipgloss.NewStyle().
				Foreground(ColorMuted)
)

// === ВКЛАДКИ (CHANNELS / DIRECT) ===
var (
	StyleTabActive = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	StyleTabInactive = lipgloss.NewStyle().
				Foreground(ColorMuted)
)

// === ЧАТ И СООБЩЕНИЯ ===
var (
	// область отображения сообщений.
	StyleChat = lipgloss.NewStyle().
			PaddingLeft(1)

	// серое время [12:01].
	StyleTimestamp = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// ник отправителя сообщения.
	StyleMessageSenderSelf = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	// ник чужого сообщения.
	StyleMessageSenderOther = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	// само тело сообщения.
	StyleMessageText = lipgloss.NewStyle().
				Foreground(ColorText)
)

// === СТАТУС ПОЛЬЗОВАТЕЛЯ (онлайн/оффлайн в DM-списке) ===
var (
	StyleOnline = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	StyleOffline = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// === ПОПАПЫ ===
var (
	StylePopup = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2)
)

// === HELP-ОКНО (вызывается по H) ===
var (
	StyleHelpPopup = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	StyleHelpTitle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	StyleHelpSection = lipgloss.NewStyle().
				Foreground(ColorPrimary)

	StyleHelpKey = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	StyleHelpDesc = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// === РАЗДЕЛИТЕЛЬ ===
var (
	StyleDivider = lipgloss.NewStyle().
		Foreground(ColorMuted)
)
