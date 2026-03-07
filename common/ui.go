package common

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"time"
)

// Version версия протокола
const Version = "1.0.0"

// ANSI escape codes
const (
	ClearLine  = "\033[2K"
	ClearAll   = "\033[2J"
	HideCursor = "\033[?25l"
	ShowCursor = "\033[?25h"
	MoveHome   = "\033[H"
	SavePos    = "\033[s"
	RestorePos = "\033[u"
)

// MoveUp перемещает курсор вверх на n строк
func MoveUp(n int) string { return fmt.Sprintf("\033[%dA", n) }

// MoveDown перемещает курсор вниз на n строк
func MoveDown(n int) string { return fmt.Sprintf("\033[%dB", n) }

// MoveRight перемещает курсор вправо на n столбцов
func MoveRight(n int) string { return fmt.Sprintf("\033[%dC", n) }

// MoveTo перемещает курсор на строку row, колонку col (1-based)
func MoveTo(row, col int) string { return fmt.Sprintf("\033[%d;%dH", row, col) }

// Colors — rich purple theme with 256-color & gradient
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"
	Blink     = "\033[5m"

	// Основные
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Яркие
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Фоновые
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"

	// SOVA purple palette (256-color)
	Purple1 = "\033[38;5;53m"  // тёмно-фиолетовый
	Purple2 = "\033[38;5;54m"  // средне-тёмный
	Purple3 = "\033[38;5;91m"  // средний фиолетовый
	Purple4 = "\033[38;5;92m"  // средне-яркий
	Purple5 = "\033[38;5;129m" // яркий фиолетовый
	Purple6 = "\033[38;5;135m" // сияющий фиолетовый
	Purple7 = "\033[38;5;141m" // лавандовый
	Purple8 = "\033[38;5;177m" // светло-лавандовый

	// Дополнительные акценты
	Gold   = "\033[38;5;220m"
	Orange = "\033[38;5;208m"
	Pink   = "\033[38;5;213m"
	Violet = "\033[38;5;99m"
	Indigo = "\033[38;5;63m"

	// Для обратной совместимости
	Purple       = "\033[35m"
	BrightPurple = "\033[95m"
)

// OwlSmall маленькая сова для inline
const OwlSmall = "{◉,◉}"

// ═══ БОЛЬШАЯ КРАСИВАЯ СОВА — анимированная, фиолетовая ═══

// Кадры летящей совы (крылья вверх / вниз)
var flyOwlUp = []string{
	`    ▓▓▓▓▓▓▓`,
	`  ╱▓ ◉   ◉ ▓╲`,
	` ╱ ▓   ▾▾  ▓ ╲`,
	`╱   ▓▓▓▓▓▓▓   ╲`,
	`     ║█ █║`,
	`      ╚═╝`,
}

var flyOwlDown = []string{
	`     ▓▓▓▓▓▓▓`,
	`   ▓ ◉   ◉ ▓`,
	`   ▓   ▾▾  ▓`,
	`    ▓▓▓▓▓▓▓`,
	` ╲   ║█ █║   ╱`,
	`  ╲   ╚═╝   ╱`,
}

var flyOwlGlide = []string{
	`     ▓▓▓▓▓▓▓`,
	`   ▓ ◉   ◉ ▓`,
	`   ▓   ▾▾  ▓`,
	`════▓▓▓▓▓▓▓════`,
	`     ║█ █║`,
	`      ╚═╝`,
}

// Большая красивая сова для баннера (сидит)
var owlSitting = []string{
	`         ▄▄▄████▄▄▄`,
	`       ▄██▀▀    ▀▀██▄`,
	`      ███  ◉    ◉  ███`,
	`      ███    ▾▾    ███`,
	`       ▀██▄▄▄▄▄▄██▀`,
	`      ╱╱ ▀████████▀ ╲╲`,
	`     ╱╱   ║██████║   ╲╲`,
	`    ▕▕    ║██████║    ▏▏`,
	`           ║║  ║║`,
	`          ▄╩╩▄▄╩╩▄`,
}

// Сова моргает
var owlBlink = []string{
	`         ▄▄▄████▄▄▄`,
	`       ▄██▀▀    ▀▀██▄`,
	`      ███  ━    ━  ███`,
	`      ███    ▾▾    ███`,
	`       ▀██▄▄▄▄▄▄██▀`,
	`      ╱╱ ▀████████▀ ╲╲`,
	`     ╱╱   ║██████║   ╲╲`,
	`    ▕▕    ║██████║    ▏▏`,
	`           ║║  ║║`,
	`          ▄╩╩▄▄╩╩▄`,
}

// Сова смотрит влево
var owlLookLeft = []string{
	`         ▄▄▄████▄▄▄`,
	`       ▄██▀▀    ▀▀██▄`,
	`      ███ ◉    ◉   ███`,
	`      ███    ▾▾    ███`,
	`       ▀██▄▄▄▄▄▄██▀`,
	`      ╱╱ ▀████████▀ ╲╲`,
	`     ╱╱   ║██████║   ╲╲`,
	`    ▕▕    ║██████║    ▏▏`,
	`           ║║  ║║`,
	`          ▄╩╩▄▄╩╩▄`,
}

// Сова смотрит вправо
var owlLookRight = []string{
	`         ▄▄▄████▄▄▄`,
	`       ▄██▀▀    ▀▀██▄`,
	`      ███   ◉    ◉ ███`,
	`      ███    ▾▾    ███`,
	`       ▀██▄▄▄▄▄▄██▀`,
	`      ╱╱ ▀████████▀ ╲╲`,
	`     ╱╱   ║██████║   ╲╲`,
	`    ▕▕    ║██████║    ▏▏`,
	`           ║║  ║║`,
	`          ▄╩╩▄▄╩╩▄`,
}

// Звёздочки/искры для следа за совой
var sparkles = []string{"✦", "✧", "⋆", "˚", "·", "∗", "⊹", "✶", "✵", "⁺"}

// Фиолетовые оттенки для градиента
var purpleGradient = []string{Purple1, Purple2, Purple3, Purple4, Purple5, Purple6, Purple7, Purple8}

// UI представляет пользовательский интерфейс
type UI struct {
	Verbose bool
}

// NewUI создает новый UI
func NewUI(verbose bool) *UI {
	EnableVTMode()
	return &UI{Verbose: verbose}
}

// colorOwlLine красит строку совы градиентом фиолетового
func colorOwlLine(line string, colorIdx int) string {
	c := purpleGradient[colorIdx%len(purpleGradient)]
	return c + Bold + line + Reset
}

// AnimateOwlFlight — главная анимация: сова ЛЕТИТ по терминалу
func (ui *UI) AnimateOwlFlight() {
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	width := 60
	owlHeight := len(flyOwlUp)
	totalFrames := width - 15

	// Выделяем строки для анимации
	for i := 0; i < owlHeight+2; i++ {
		fmt.Println()
	}

	// Массив для "следа" из звёздочек
	type trailStar struct {
		x, y int
		life int
		ch   string
	}
	var trail []trailStar

	for frame := 0; frame < totalFrames; frame++ {
		// Перемещаемся вверх на всю область анимации
		fmt.Print(MoveUp(owlHeight + 2))

		// Выбираем кадр совы (машет крыльями)
		var owlFrame []string
		switch frame % 6 {
		case 0, 1:
			owlFrame = flyOwlUp
		case 2:
			owlFrame = flyOwlGlide
		case 3, 4:
			owlFrame = flyOwlDown
		case 5:
			owlFrame = flyOwlGlide
		}

		// Вертикальное покачивание
		yOff := 0
		if frame%4 < 2 {
			yOff = 0
		} else {
			yOff = 1
		}

		// Добавляем звёздочку в след
		if frame > 2 && frame%2 == 0 {
			trail = append(trail, trailStar{
				x:    frame - 2,
				y:    owlHeight/2 + yOff,
				life: 6 + rand.Intn(4),
				ch:   sparkles[rand.Intn(len(sparkles))],
			})
		}

		// Рисуем каждую строку
		for row := 0; row < owlHeight+2; row++ {
			fmt.Print(ClearLine)
			line := ""

			// Рисуем звёздочки следа
			lineRunes := make([]byte, width+20)
			for i := range lineRunes {
				lineRunes[i] = ' '
			}

			for _, s := range trail {
				if s.y == row && s.x >= 0 && s.x < len(lineRunes) {
					lineRunes[s.x] = '*' // placeholder
				}
			}

			// Рисуем строку совы
			owlRow := row - yOff
			if owlRow >= 0 && owlRow < len(owlFrame) {
				padding := strings.Repeat(" ", frame+2)
				gradIdx := owlRow
				line = padding + colorOwlLine(owlFrame[owlRow], gradIdx)
			}

			if line == "" {
				// Рисуем только звёздочки
				starLine := ""
				for _, s := range trail {
					if s.y == row {
						pad := s.x
						if pad > len(starLine) {
							starLine += strings.Repeat(" ", pad-len(starLine))
						}
						// Цвет звезды зависит от оставшейся жизни
						c := purpleGradient[(s.life)%len(purpleGradient)]
						starLine += c + s.ch + Reset
					}
				}
				if starLine != "" {
					line = starLine
				}
			}

			fmt.Println(line)
		}

		// Стареем звёздочки
		alive := trail[:0]
		for _, s := range trail {
			s.life--
			if s.life > 0 {
				alive = append(alive, s)
			}
		}
		trail = alive

		time.Sleep(55 * time.Millisecond)
	}

	// Очищаем область анимации полёта
	fmt.Print(MoveUp(owlHeight + 2))
	for i := 0; i < owlHeight+2; i++ {
		fmt.Print(ClearLine)
		fmt.Println()
	}
	fmt.Print(MoveUp(owlHeight + 2))
}

// AnimateOwlLanding — сова приземляется и моргает
func (ui *UI) AnimateOwlLanding() {
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	owlHeight := len(owlSitting)

	// Появление совы построчно с фиолетовым градиентом
	for i, line := range owlSitting {
		gradIdx := i % len(purpleGradient)
		fmt.Println("  " + purpleGradient[gradIdx] + Bold + line + Reset)
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	// Моргание 2 раза
	for blink := 0; blink < 2; blink++ {
		ui.renderOwlFrame2(owlBlink, owlHeight)
		time.Sleep(120 * time.Millisecond)
		ui.renderOwlFrame2(owlSitting, owlHeight)
		time.Sleep(350 * time.Millisecond)
	}

	// Посмотреть по сторонам
	ui.renderOwlFrame2(owlLookLeft, owlHeight)
	time.Sleep(350 * time.Millisecond)
	ui.renderOwlFrame2(owlLookRight, owlHeight)
	time.Sleep(350 * time.Millisecond)
	ui.renderOwlFrame2(owlSitting, owlHeight)
	time.Sleep(150 * time.Millisecond)
}

// renderOwlFrame2 перерисовывает сидящую сову на месте
func (ui *UI) renderOwlFrame2(frame []string, lineCount int) {
	fmt.Print(MoveUp(lineCount))
	for i, line := range frame {
		fmt.Print(ClearLine)
		gradIdx := i % len(purpleGradient)
		fmt.Println("  " + purpleGradient[gradIdx] + Bold + line + Reset)
	}
}

// PrintBanner печатает баннер с полной анимацией: полёт → приземление → баннер
func (ui *UI) PrintBanner() {
	fmt.Println()

	// Фаза 1: Сова летит по терминалу
	ui.AnimateOwlFlight()

	// Фаза 2: Сова приземляется и моргает
	ui.AnimateOwlLanding()

	// Фаза 3: Фиолетовый баннер
	fmt.Println()
	ui.printPurpleBox()
	fmt.Println()
}

// printPurpleBox — красивый бокс с градиентом
func (ui *UI) printPurpleBox() {
	top := "  ╔════════════════════════════════════════════════════╗"
	mid1 := "  ║     🦉  S O V A   P r o t o c o l   v" + Version + "       ║"
	mid2 := "  ║     Secure Obfuscated Versatile Adapter          ║"
	mid3 := "  ║     AI-Powered · Post-Quantum · Free             ║"
	bot := "  ╚════════════════════════════════════════════════════╝"

	fmt.Println(Purple6 + Bold + top + Reset)
	fmt.Println(Purple7 + Bold + mid1 + Reset)
	fmt.Println(Purple5 + mid2 + Reset)
	fmt.Println(Purple4 + mid3 + Reset)
	fmt.Println(Purple6 + Bold + bot + Reset)

	fmt.Println()
	fmt.Printf("  %s%s %s/%s • Go %s • PQ Crypto • %s%s\n",
		Purple8, OwlSmall, runtime.GOOS, runtime.GOARCH, runtime.Version(), Version, Reset)
	fmt.Printf("  %s%s%s\n", Purple3, strings.Repeat("━", 54), Reset)
}

// PrintBannerQuiet печатает баннер без анимации — только сидящая сова + бокс
func (ui *UI) PrintBannerQuiet() {
	fmt.Println()
	for i, line := range owlSitting {
		gradIdx := i % len(purpleGradient)
		fmt.Println("  " + purpleGradient[gradIdx] + Bold + line + Reset)
	}
	fmt.Println()
	ui.printPurpleBox()
	fmt.Println()
}

// PrintStatus печатает статус с таймстампом
func (ui *UI) PrintStatus(status string, color string) {
	fmt.Printf("  %s▸%s %s[%s]%s %s%s%s\n",
		Purple6, Reset, Dim, time.Now().Format("15:04:05"), Reset, color, status, Reset)
}

// PrintProgress печатает прогресс-бар
func (ui *UI) PrintProgress(current, total int, message string) {
	percentage := float64(current) / float64(total) * 100
	barWidth := 40
	filled := int(percentage / 100 * float64(barWidth))
	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < barWidth; i++ {
		bar += "░"
	}
	fmt.Printf("\r  %s[%.1f%%]%s %s%s%s %s%s",
		Purple6, percentage, Reset, Purple7, bar, Reset, Dim+message, Reset)
	if current == total {
		fmt.Println()
	}
}

// PrintError печатает ошибку
func (ui *UI) PrintError(err error) {
	fmt.Printf("  %s✗ [ERROR]%s %v\n", Red+Bold, Reset, err)
}

// PrintSuccess печатает успех
func (ui *UI) PrintSuccess(message string) {
	fmt.Printf("  %s✓%s %s\n", BrightGreen+Bold, Reset, message)
}

// PrintInfo печатает информацию
func (ui *UI) PrintInfo(message string) {
	if ui.Verbose {
		fmt.Printf("  %s○%s %s%s%s\n", Purple7, Reset, Dim, message, Reset)
	}
}

// PrintInfoAlways печатает информацию всегда (даже без verbose)
func (ui *UI) PrintInfoAlways(message string) {
	fmt.Printf("  %s○%s %s\n", Purple7, Reset, message)
}

// PrintWarning печатает предупреждение
func (ui *UI) PrintWarning(message string) {
	fmt.Printf("  %s⚠ [WARN]%s %s\n", Yellow+Bold, Reset, message)
}

// PrintSection печатает заголовок секции
func (ui *UI) PrintSection(title string) {
	fmt.Println()
	fmt.Printf("  %s━━━ %s%s %s━━━%s\n", Purple5, Purple7+Bold, title, Purple5, Reset)
}

// PrintKeyValue печатает ключ-значение
func (ui *UI) PrintKeyValue(key, value string) {
	fmt.Printf("  %s│%s %s%-24s%s %s\n", Purple5, Reset, Purple8, key, Reset, value)
}

// AnimateConnection анимирует подключение с красивым спиннером
func (ui *UI) AnimateConnection() {
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	msgs := []string{
		"Initializing encrypted tunnel...",
		"Negotiating PQ key exchange...",
		"Establishing secure channel...",
		"Verifying zero-knowledge proof...",
		"Tunnel active!",
	}
	for i := 0; i < 25; i++ {
		msg := msgs[i*len(msgs)/25]
		fmt.Printf("\r  %s%s%s %s%s%s",
			Purple6+Bold, frames[i%len(frames)], Reset,
			Purple8, msg, Reset)
		time.Sleep(80 * time.Millisecond)
	}
	fmt.Printf("\r  %s✓ Encrypted tunnel established!                              %s\n", BrightGreen+Bold, Reset)
}

// AnimateLoading анимирует загрузку с сообщением
func (ui *UI) AnimateLoading(message string, duration time.Duration) {
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	frames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	start := time.Now()
	i := 0
	for time.Since(start) < duration {
		c := purpleGradient[i%len(purpleGradient)]
		fmt.Printf("\r  %s%s%s %s%s%s", c+Bold, frames[i%len(frames)], Reset, Purple8, message, Reset)
		time.Sleep(80 * time.Millisecond)
		i++
	}
	fmt.Printf("\r  %s✓%s %s%s\n", BrightGreen+Bold, Reset, message, strings.Repeat(" ", 10))
}

// AnimateOwlThinking — сова "думает" (моргает пока идёт операция)
func (ui *UI) AnimateOwlThinking(message string, done <-chan struct{}) {
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	eyes := []string{"◉", "○", "◉", "━"}
	i := 0
	for {
		select {
		case <-done:
			fmt.Printf("\r  %s%s%s %s✓ %s%s\n", Purple6, OwlSmall, Reset, BrightGreen+Bold, Reset+message, Reset)
			return
		default:
			owl := fmt.Sprintf("{%s,%s}", eyes[i%len(eyes)], eyes[(i+1)%len(eyes)])
			c := purpleGradient[i%len(purpleGradient)]
			fmt.Printf("\r  %s%s%s %s%s%s", c+Bold, owl, Reset, Purple8, message, Reset)
			time.Sleep(200 * time.Millisecond)
			i++
		}
	}
}

// PrintSystemInfo печатает системную информацию
func (ui *UI) PrintSystemInfo() {
	ui.PrintSection("System Information")
	ui.PrintKeyValue("OS:", runtime.GOOS)
	ui.PrintKeyValue("Architecture:", runtime.GOARCH)
	ui.PrintKeyValue("Go Runtime:", runtime.Version())
	ui.PrintKeyValue("CPU Cores:", fmt.Sprintf("%d", runtime.NumCPU()))
	ui.PrintKeyValue("Goroutines:", fmt.Sprintf("%d", runtime.NumGoroutine()))
	ui.PrintKeyValue("SOVA Version:", Version)

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	ui.PrintKeyValue("Memory (alloc):", formatBytesUI(int64(mem.Alloc)))
	ui.PrintKeyValue("Memory (sys):", formatBytesUI(int64(mem.Sys)))
	ui.PrintKeyValue("GC Runs:", fmt.Sprintf("%d", mem.NumGC))
	fmt.Println()
}

// PrintConfig печатает текущую конфигурацию
func (ui *UI) PrintConfig(cfg *Config) {
	ui.PrintSection("Configuration")
	ui.PrintKeyValue("Mode:", cfg.Mode)
	ui.PrintKeyValue("Listen:", cfg.ListenAddress())
	if cfg.ServerAddr != "" {
		ui.PrintKeyValue("Server:", cfg.ServerAddress())
	}
	ui.PrintKeyValue("Encryption:", cfg.Encryption.Algorithm)
	ui.PrintKeyValue("PQ Crypto:", boolToStatus(cfg.Encryption.PQEnabled))
	ui.PrintKeyValue("ZKP Auth:", boolToStatus(cfg.Encryption.ZKPEnabled))
	ui.PrintKeyValue("Stealth:", boolToStatus(cfg.Stealth.Enabled))
	ui.PrintKeyValue("Stealth Profile:", cfg.Stealth.Profile)
	ui.PrintKeyValue("TLS Fingerprint:", cfg.Stealth.TLSFingerprint)
	ui.PrintKeyValue("AI Adapter:", boolToStatus(cfg.Features.AIAdapter))
	ui.PrintKeyValue("Compression:", boolToStatus(cfg.Features.Compression))
	ui.PrintKeyValue("Smart Routing:", boolToStatus(cfg.Features.SmartRouting))
	ui.PrintKeyValue("Connection Pool:", boolToStatus(cfg.Features.ConnectionPool))
	ui.PrintKeyValue("DNS-over-SOVA:", boolToStatus(cfg.DNS.Enabled))
	if cfg.DNS.Enabled {
		ui.PrintKeyValue("DNS Port:", fmt.Sprintf("%d", cfg.DNS.Port))
		ui.PrintKeyValue("DNS Upstream:", cfg.DNS.Upstream)
	}
	ui.PrintKeyValue("Mesh Network:", boolToStatus(cfg.Features.MeshNetwork))
	ui.PrintKeyValue("Offline First:", boolToStatus(cfg.Features.OfflineFirst))
	ui.PrintKeyValue("API:", boolToStatus(cfg.API.Enabled))
	if cfg.API.Enabled {
		ui.PrintKeyValue("API Address:", fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port))
	}
	ui.PrintKeyValue("Dashboard:", boolToStatus(cfg.Features.Dashboard))
	ui.PrintKeyValue("Transport:", cfg.Transport.Mode)
	ui.PrintKeyValue("Log Level:", cfg.LogLevel)
	fmt.Println()
}

// PrintFeatures печатает статус всех модулей в табличном виде
func (ui *UI) PrintFeatures(cfg *Config) {
	ui.PrintSection("Modules Status")

	type feat struct {
		name  string
		on    bool
		descr string
	}

	features := []feat{
		{"pq_crypto", cfg.Encryption.PQEnabled, "Post-Quantum Kyber1024 + Dilithium5"},
		{"zkp", cfg.Encryption.ZKPEnabled, "Zero-Knowledge Proof Auth"},
		{"stealth", cfg.Stealth.Enabled, "Traffic mimicry & obfuscation"},
		{"padding", cfg.Stealth.PaddingEnabled, "Intelligent packet padding"},
		{"decoy", cfg.Stealth.DecoyEnabled, "Decoy background traffic"},
		{"ai_adapter", cfg.Features.AIAdapter, "AI adaptive DPI bypass"},
		{"compression", cfg.Features.Compression, "Gzip traffic compression"},
		{"connection_pool", cfg.Features.ConnectionPool, "Connection reuse pool"},
		{"smart_routing", cfg.Features.SmartRouting, "Latency-based route optimizer"},
		{"mesh_network", cfg.Features.MeshNetwork, "Peer-to-peer mesh networking"},
		{"offline_first", cfg.Features.OfflineFirst, "Offline cache & sync"},
		{"dns", cfg.DNS.Enabled, "DNS-over-SOVA resolver"},
		{"api", cfg.API.Enabled, "Management REST API"},
		{"dashboard", cfg.Features.Dashboard, "Web dashboard"},
		{"auto_proxy", cfg.Features.AutoProxy, "Auto system proxy config"},
	}

	for _, f := range features {
		status := BrightGreen + Bold + " ON " + Reset
		if !f.on {
			status = Red + "OFF " + Reset
		}
		fmt.Printf("  %s[%s]%s %s%-18s%s %s%s%s\n",
			Purple5, status, Reset,
			Purple8+Bold, f.name, Reset,
			Dim, f.descr, Reset)
	}
	fmt.Println()
}

// PrintHelp печатает справку по командам
func (ui *UI) PrintHelp() {
	ui.PrintSection("Commands")
	cmds := []struct{ cmd, desc string }{
		{"sova", "Start SOVA tunnel (local SOCKS5 proxy)"},
		{"sova start", "Same as above"},
		{"sova connect <server>", "Connect through remote SOVA server"},
		{"sova config", "Show current configuration"},
		{"sova config set <k> <v>", "Update config setting"},
		{"sova config reset", "Reset config to defaults"},
		{"sova config json", "Export config as JSON"},
		{"sova config path", "Show config file path"},
		{"sova features", "Show all modules status"},
		{"sova enable <module>", "Enable a module"},
		{"sova disable <module>", "Disable a module"},
		{"sova status", "Show tunnel status, stats, system info"},
		{"sova profiles", "List config profiles"},
		{"sova profile <name>", "Switch to config profile"},
		{"sova logs", "Show recent log entries"},
		{"sova bench", "Run quick network benchmark"},
		{"sova help", "Show this help"},
		{"sova version", "Show version info"},
	}
	for _, c := range cmds {
		fmt.Printf("  %s%-28s%s %s%s%s\n", Purple7+Bold, c.cmd, Reset, Dim, c.desc, Reset)
	}

	ui.PrintSection("Config Keys")
	keys := []struct{ key, desc string }{
		{"mode", "local | remote | server"},
		{"listen_addr", "Proxy listen address (127.0.0.1)"},
		{"listen_port", "Proxy listen port (1080)"},
		{"server_addr", "Remote server address"},
		{"server_port", "Remote server port (443)"},
		{"encryption", "aes-256-gcm | chacha20-poly1305"},
		{"stealth_profile", "chrome | youtube | cloud_api | random"},
		{"tls_fingerprint", "chrome | firefox | safari | random"},
		{"transport_mode", "auto | web_mirror | quic | websocket"},
		{"log_level", "debug | info | warn | error"},
		{"api_port", "Management API port (8080)"},
		{"dns_upstream", "DNS upstream (8.8.8.8:53)"},
		{"dns_port", "DNS listen port (5353)"},
		{"jitter_ms", "Stealth jitter in ms (50)"},
	}
	for _, k := range keys {
		fmt.Printf("  %s%-20s%s %s%s%s\n", Purple7, k.key, Reset, Dim, k.desc, Reset)
	}

	ui.PrintSection("Toggleable Modules")
	modules := []string{
		"pq_crypto, zkp, stealth, padding, decoy, ai_adapter,",
		"compression, connection_pool, smart_routing, mesh_network,",
		"offline_first, dns, api, dashboard, auto_proxy",
	}
	for _, m := range modules {
		fmt.Printf("  %s%s%s\n", Purple8, m, Reset)
	}

	ui.PrintSection("Proxy Setup")
	fmt.Printf("  %sAfter starting SOVA, configure your browser/system proxy:%s\n", Dim, Reset)
	fmt.Printf("  %sSOCKS5 → 127.0.0.1:1080%s\n", Gold+Bold, Reset)
	fmt.Printf("  %sOr use: %scurl --proxy socks5h://127.0.0.1:1080 https://youtube.com%s\n", Dim, Yellow, Reset)

	ui.PrintSection("API Endpoints")
	fmt.Printf("  %sGET  /api/status      %s— System status%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sGET  /api/health      %s— Health check%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sGET  /api/config      %s— Full configuration%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sPUT  /api/config      %s— Update full config%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sPOST /api/config/set  %s— Set single value%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sPOST /api/config/reset%s— Reset to defaults%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sGET  /api/features    %s— All modules status%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sPOST /api/feature/    %s— Toggle module%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sGET  /api/system      %s— System info (CPU/RAM)%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sGET  /api/stats       %s— Traffic statistics%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sGET  /api/logs        %s— Recent log entries%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sGET  /api/profiles    %s— Config profiles%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sPOST /api/profile     %s— Switch profile%s\n", Purple7, Dim, Reset)
	fmt.Printf("  %sPOST /api/restart     %s— Restart tunnel%s\n", Purple7, Dim, Reset)
	fmt.Println()
}

// ConfirmAction запрашивает подтверждение
func (ui *UI) ConfirmAction(prompt string) bool {
	fmt.Printf("  %s? %s (y/N): %s", Gold, prompt, Reset)
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y"
}

// ExitWithError выходит с ошибкой
func (ui *UI) ExitWithError(err error) {
	ui.PrintError(err)
	os.Exit(1)
}

// PrintDivider печатает разделитель
func (ui *UI) PrintDivider() {
	fmt.Printf("  %s%s%s\n", Purple3, strings.Repeat("━", 54), Reset)
}

// PrintTunnelActive печатает финальное сообщение об активном туннеле
func (ui *UI) PrintTunnelActive(listenAddr string, cfg *Config) {
	ui.PrintSection("🦉 SOVA Tunnel Active")
	ui.PrintKeyValue("SOCKS5 Proxy:", Gold+Bold+listenAddr+Reset)
	ui.PrintKeyValue("Protocol:", "SOVA v"+Version+" (PQ-encrypted)")
	ui.PrintKeyValue("Encryption:", cfg.Encryption.Algorithm)
	ui.PrintKeyValue("Stealth:", cfg.Stealth.Profile)
	if cfg.API.Enabled {
		ui.PrintKeyValue("API:", fmt.Sprintf("http://%s:%d/api/", cfg.API.Host, cfg.API.Port))
	}
	fmt.Println()
	fmt.Printf("  %sConfigure your browser or system proxy:%s\n", Dim, Reset)
	fmt.Printf("  %s→ SOCKS5 Host: %s  Port: %d%s\n", Gold+Bold, cfg.ListenAddr, cfg.ListenPort, Reset)
	fmt.Printf("  %s→ curl --proxy socks5h://%s https://youtube.com%s\n", Dim, listenAddr, Reset)
	fmt.Println()
	ui.PrintDivider()
	fmt.Printf("  %sPress Ctrl+C to stop SOVA%s\n", Dim, Reset)
	fmt.Println()
}

func boolToStatus(b bool) string {
	if b {
		return BrightGreen + Bold + "enabled" + Reset
	}
	return Red + "disabled" + Reset
}

func formatBytesUI(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
