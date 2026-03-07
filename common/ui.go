package common

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// Version версия протокола
const Version = "1.0.0"

// Owl animation frames — сова моргает, крутит головой, машет крыльями
var OwlFrames = []string{
	// Frame 0: глаза открыты
	`
        ▄▄▄▄▄▄▄▄▄▄▄
       ▐  ◉      ◉  ▌
       ▐     ▼▼     ▌
        ▀▄▄▄▄▄▄▄▄▀
         ╱╱    ╲╲
        ╱╱  ██  ╲╲
       ▕▕   ██   ▏▏
`,
	// Frame 1: моргает
	`
        ▄▄▄▄▄▄▄▄▄▄▄
       ▐  ─      ─  ▌
       ▐     ▼▼     ▌
        ▀▄▄▄▄▄▄▄▄▀
         ╱╱    ╲╲
        ╱╱  ██  ╲╲
       ▕▕   ██   ▏▏
`,
	// Frame 2: смотрит влево
	`
        ▄▄▄▄▄▄▄▄▄▄▄
       ▐ ◉      ◉   ▌
       ▐     ▼▼     ▌
        ▀▄▄▄▄▄▄▄▄▀
         ╱╱    ╲╲
        ╱╱  ██  ╲╲
       ▕▕   ██   ▏▏
`,
	// Frame 3: смотрит вправо
	`
        ▄▄▄▄▄▄▄▄▄▄▄
       ▐   ◉      ◉▌
       ▐     ▼▼     ▌
        ▀▄▄▄▄▄▄▄▄▀
         ╱╱    ╲╲
        ╱╱  ██  ╲╲
       ▕▕   ██   ▏▏
`,
	// Frame 4: крылья расправлены
	`
        ▄▄▄▄▄▄▄▄▄▄▄
       ▐  ◉      ◉  ▌
       ▐     ▼▼     ▌
        ▀▄▄▄▄▄▄▄▄▀
      ╱╱╱╱    ╲╲╲╲
     ╱╱╱╱  ██  ╲╲╲╲
       ▕▕   ██   ▏▏
`,
	// Frame 5: крылья вверх
	`
     ╲  ▄▄▄▄▄▄▄▄▄▄▄  ╱
      ▐  ◉      ◉  ▌
      ▐     ▼▼     ▌
       ▀▄▄▄▄▄▄▄▄▀
         ╱╱    ╲╲
        ╱╱  ██  ╲╲
       ▕▕   ██   ▏▏
`,
}

// OwlSmall маленькая сова для inline
const OwlSmall = "{◉,◉}"

// ClearLine ANSI escape для очистки строки
const ClearLine = "\033[2K"

// MoveUp перемещает курсор вверх на n строк
func MoveUp(n int) string {
	return fmt.Sprintf("\033[%dA", n)
}

// HideCursor / ShowCursor
const HideCursor = "\033[?25l"
const ShowCursor = "\033[?25h"

// Colors — purple theme
const (
	Purple       = "\033[35m"
	BrightPurple = "\033[95m"
	Reset        = "\033[0m"
	Cyan         = "\033[36m"
	BrightCyan   = "\033[96m"
	Green        = "\033[32m"
	BrightGreen  = "\033[92m"
	Yellow       = "\033[33m"
	Red          = "\033[31m"
	White        = "\033[37m"
	Bold         = "\033[1m"
	Dim          = "\033[2m"
	BgPurple     = "\033[45m"
	Underline    = "\033[4m"
)

// UI представляет пользовательский интерфейс
type UI struct {
	Verbose bool
}

// NewUI создает новый UI
func NewUI(verbose bool) *UI {
	return &UI{Verbose: verbose}
}

// AnimateOwlStartup — главная анимация совы при запуске
func (ui *UI) AnimateOwlStartup() {
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	// Фаза 1: Появление совы построчно (typewriter)
	owlLines := strings.Split(OwlFrames[0], "\n")
	for _, line := range owlLines {
		if line == "" {
			continue
		}
		fmt.Println(BrightPurple + line + Reset)
		time.Sleep(60 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	lineCount := len(owlLines) - 1 // кол-во напечатанных строк

	// Фаза 2: Сова моргает 2 раза
	for blink := 0; blink < 2; blink++ {
		ui.renderOwlFrame(OwlFrames[1], lineCount) // глаза закрыты
		time.Sleep(150 * time.Millisecond)
		ui.renderOwlFrame(OwlFrames[0], lineCount) // глаза открыты
		time.Sleep(300 * time.Millisecond)
	}

	// Фаза 3: Сова смотрит по сторонам
	ui.renderOwlFrame(OwlFrames[2], lineCount) // влево
	time.Sleep(400 * time.Millisecond)
	ui.renderOwlFrame(OwlFrames[3], lineCount) // вправо
	time.Sleep(400 * time.Millisecond)
	ui.renderOwlFrame(OwlFrames[0], lineCount) // прямо
	time.Sleep(200 * time.Millisecond)

	// Фаза 4: Машет крыльями
	for flap := 0; flap < 3; flap++ {
		ui.renderOwlFrame(OwlFrames[4], lineCount) // крылья раскрыты
		time.Sleep(120 * time.Millisecond)
		ui.renderOwlFrame(OwlFrames[5], lineCount) // крылья вверх
		time.Sleep(120 * time.Millisecond)
	}
	ui.renderOwlFrame(OwlFrames[0], lineCount) // нормальная поза
	time.Sleep(200 * time.Millisecond)
}

// renderOwlFrame перерисовывает сову на месте (перезаписывая предыдущий кадр)
func (ui *UI) renderOwlFrame(frame string, lineCount int) {
	// Перемещаемся вверх на lineCount строк
	fmt.Print(MoveUp(lineCount))
	lines := strings.Split(frame, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fmt.Print(ClearLine)
		fmt.Println(BrightPurple + line + Reset)
	}
}

// PrintBanner печатает баннер с анимированной совой
func (ui *UI) PrintBanner() {
	fmt.Println()

	// Анимированная сова
	ui.AnimateOwlStartup()

	// Текстовый баннер
	fmt.Println()
	fmt.Println(Bold + BrightPurple + "  ╔══════════════════════════════════════════════╗" + Reset)
	fmt.Println(Bold + BrightPurple + "  ║         SOVA Protocol v" + Version + "                  ║" + Reset)
	fmt.Println(Bold + BrightPurple + "  ║   Secure Obfuscated Versatile Adapter        ║" + Reset)
	fmt.Println(Bold + BrightPurple + "  ╚══════════════════════════════════════════════╝" + Reset)
	fmt.Println(Dim + Cyan + "  AI-Powered Autonomous Protocol for Internet Freedom" + Reset)
	fmt.Println(Dim + Purple + "  " + strings.Repeat("─", 48) + Reset)
	fmt.Printf("  %s%s %s/%s | Go %s | PQ Crypto%s\n", Dim, OwlSmall, runtime.GOOS, runtime.GOARCH, runtime.Version(), Reset)
	fmt.Println()
}

// PrintBannerQuiet печатает баннер без анимации
func (ui *UI) PrintBannerQuiet() {
	fmt.Println()
	fmt.Println(BrightPurple + OwlFrames[0] + Reset)
	fmt.Println(Bold + BrightPurple + "  SOVA Protocol v" + Version + Reset)
	fmt.Println(Cyan + "  Secure Obfuscated Versatile Adapter" + Reset)
	fmt.Println(Dim + Purple + "  " + strings.Repeat("─", 48) + Reset)
	fmt.Printf("  %s%s %s/%s | Go %s%s\n", Dim, OwlSmall, runtime.GOOS, runtime.GOARCH, runtime.Version(), Reset)
	fmt.Println()
}

// PrintStatus печатает статус с таймстампом
func (ui *UI) PrintStatus(status string, color string) {
	fmt.Printf("%s  ▸ [%s] %s%s\n", color, time.Now().Format("15:04:05"), status, Reset)
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
	fmt.Printf("\r%s  [%.1f%%] %s %s", Purple, percentage, bar, message+Reset)
	if current == total {
		fmt.Println()
	}
}

// PrintError печатает ошибку
func (ui *UI) PrintError(err error) {
	fmt.Printf("%s  ✗ [ERROR] %v%s\n", Red, err, Reset)
}

// PrintSuccess печатает успех
func (ui *UI) PrintSuccess(message string) {
	fmt.Printf("%s  ✓ %s%s\n", BrightGreen, message, Reset)
}

// PrintInfo печатает информацию
func (ui *UI) PrintInfo(message string) {
	if ui.Verbose {
		fmt.Printf("%s  ○ %s%s\n", Cyan, message, Reset)
	}
}

// PrintInfoAlways печатает информацию всегда (даже без verbose)
func (ui *UI) PrintInfoAlways(message string) {
	fmt.Printf("%s  ○ %s%s\n", Cyan, message, Reset)
}

// PrintWarning печатает предупреждение
func (ui *UI) PrintWarning(message string) {
	fmt.Printf("%s  ⚠ [WARN] %s%s\n", Yellow, message, Reset)
}

// PrintSection печатает заголовок секции
func (ui *UI) PrintSection(title string) {
	fmt.Println()
	fmt.Printf("%s%s  ═══ %s ═══%s\n", Bold, BrightPurple, title, Reset)
}

// PrintKeyValue печатает ключ-значение
func (ui *UI) PrintKeyValue(key, value string) {
	fmt.Printf("%s  │ %-24s%s %s%s\n", Dim+Purple, key, Reset, value, Reset)
}

// AnimateConnection анимирует подключение
func (ui *UI) AnimateConnection() {
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for i := 0; i < 20; i++ {
		fmt.Printf("\r%s  %s %sУстановка защищённого туннеля...%s", Purple, frames[i%len(frames)], Yellow, Reset)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Printf("\r%s  ✓ Защищённый туннель установлен!              %s\n", BrightGreen, Reset)
}

// AnimateLoading анимирует загрузку с сообщением
func (ui *UI) AnimateLoading(message string, duration time.Duration) {
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	frames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	start := time.Now()
	i := 0
	for time.Since(start) < duration {
		fmt.Printf("\r%s  %s %s%s%s", BrightPurple, frames[i%len(frames)], Cyan, message, Reset)
		time.Sleep(80 * time.Millisecond)
		i++
	}
	fmt.Printf("\r%s  ✓ %s%s\n", BrightGreen, message, Reset)
}

// AnimateOwlThinking — сова "думает" (моргает пока идёт операция)
func (ui *UI) AnimateOwlThinking(message string, done <-chan struct{}) {
	fmt.Print(HideCursor)
	defer fmt.Print(ShowCursor)

	eyes := []string{"◉", "○", "◉", "─"}
	i := 0
	for {
		select {
		case <-done:
			fmt.Printf("\r%s  %s ✓ %s%s\n", BrightPurple, OwlSmall, BrightGreen+message, Reset)
			return
		default:
			owl := fmt.Sprintf("{%s,%s}", eyes[i%len(eyes)], eyes[(i+1)%len(eyes)])
			fmt.Printf("\r%s  %s %s%s%s", BrightPurple, owl, Cyan, message, Reset)
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
	ui.PrintKeyValue("AI Adapter:", boolToStatus(cfg.Features.AIAdapter))
	ui.PrintKeyValue("Compression:", boolToStatus(cfg.Features.Compression))
	ui.PrintKeyValue("Smart Routing:", boolToStatus(cfg.Features.SmartRouting))
	ui.PrintKeyValue("DNS-over-SOVA:", boolToStatus(cfg.DNS.Enabled))
	ui.PrintKeyValue("API:", boolToStatus(cfg.API.Enabled))
	if cfg.API.Enabled {
		ui.PrintKeyValue("API Address:", fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port))
	}
	ui.PrintKeyValue("Dashboard:", boolToStatus(cfg.Features.Dashboard))
	fmt.Println()
}

// PrintFeatures печатает статус всех модулей
func (ui *UI) PrintFeatures(cfg *Config) {
	ui.PrintSection("Modules")
	features := cfg.GetFeatureStatus()
	for name, enabled := range features {
		status := BrightGreen + "ON " + Reset
		if !enabled {
			status = Red + "OFF" + Reset
		}
		fmt.Printf("  %s[%s]%s  %s\n", Dim, status, Reset, name)
	}
	fmt.Println()
}

// PrintHelp печатает справку по командам
func (ui *UI) PrintHelp() {
	ui.PrintSection("Commands")
	fmt.Println(Cyan + "  sova" + Reset + "                       Start SOVA tunnel (local SOCKS5 proxy)")
	fmt.Println(Cyan + "  sova start" + Reset + "                 Same as above")
	fmt.Println(Cyan + "  sova connect <server>" + Reset + "      Connect through remote SOVA server")
	fmt.Println(Cyan + "  sova config" + Reset + "                Show current configuration")
	fmt.Println(Cyan + "  sova config set <k> <v>" + Reset + "    Update config setting")
	fmt.Println(Cyan + "  sova features" + Reset + "              Show all modules status")
	fmt.Println(Cyan + "  sova enable <module>" + Reset + "       Enable a module")
	fmt.Println(Cyan + "  sova disable <module>" + Reset + "      Disable a module")
	fmt.Println(Cyan + "  sova status" + Reset + "                Show tunnel status and stats")
	fmt.Println(Cyan + "  sova help" + Reset + "                  Show this help")
	fmt.Println(Cyan + "  sova version" + Reset + "               Show version info")
	fmt.Println()
	ui.PrintSection("Proxy Setup")
	fmt.Println(Dim + "  After starting SOVA, configure your browser/system proxy:" + Reset)
	fmt.Println(Yellow + "  SOCKS5 → 127.0.0.1:1080" + Reset)
	fmt.Println(Dim + "  Or use with curl:" + Reset)
	fmt.Println(Yellow + "  curl --proxy socks5h://127.0.0.1:1080 https://youtube.com" + Reset)
	fmt.Println()
}

// ConfirmAction запрашивает подтверждение
func (ui *UI) ConfirmAction(prompt string) bool {
	fmt.Printf("%s  ? %s (y/N): %s", Yellow, prompt, Reset)
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
	fmt.Printf("%s  %s%s\n", Dim+Purple, strings.Repeat("─", 48), Reset)
}

func boolToStatus(b bool) string {
	if b {
		return BrightGreen + "enabled" + Reset
	}
	return Red + "disabled" + Reset
}
