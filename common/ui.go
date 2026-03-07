package common

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// Version версия протокола
const Version = "2.0.0"

// Owl ASCII art
const OwlIcon = `
    ┌─────────────────┐
    │    ,___,      │
    │    {o,o}      │
    │    /)  )      │
    │    -"  "-     │
    │   SOVA AI     │
    └─────────────────┘
`

// OwlSmall маленькая сова
const OwlSmall = `{o,o}`

// Colors for purple theme
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

// PrintBanner печатает баннер с совой
func (ui *UI) PrintBanner() {
	fmt.Println()
	fmt.Print(BrightPurple + OwlIcon + Reset)
	fmt.Println(Bold + BrightPurple + "  SOVA Protocol v" + Version + Reset)
	fmt.Println(Cyan + "  Secure Obfuscated Versatile Adapter" + Reset)
	fmt.Println(Dim + Purple + "  AI-Powered Autonomous Protocol for Internet Survival" + Reset)
	fmt.Println(Dim + Purple + strings.Repeat("─", 50) + Reset)
	fmt.Printf("  %s%s %s/%s | Go %s%s\n", Dim, OwlSmall, runtime.GOOS, runtime.GOARCH, runtime.Version(), Reset)
	fmt.Println()
}

// PrintStatus печатает статус
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
	fmt.Printf("%s  │ %-20s%s %s%s\n", Dim+Purple, key, Reset, value, Reset)
}

// AnimateConnection анимирует подключение
func (ui *UI) AnimateConnection() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for i := 0; i < 15; i++ {
		fmt.Printf("\r%s  %s %sПодключение к SOVA...%s", Purple, frames[i%len(frames)], Yellow, Reset)
		time.Sleep(150 * time.Millisecond)
	}
	fmt.Printf("\r%s  ✓ Подключено к SOVA!              %s\n", BrightGreen, Reset)
}

// AnimateLoading анимирует загрузку
func (ui *UI) AnimateLoading(message string, duration time.Duration) {
	frames := []string{"⠀", "⠁", "⠃", "⠇", "⠏", "⠟", "⠿", "⡿", "⣿", "⣾", "⣼", "⣸", "⣰", "⣠", "⣀", "⢀"}
	start := time.Now()
	i := 0
	for time.Since(start) < duration {
		fmt.Printf("\r%s  %s %s%s%s", BrightPurple, frames[i%len(frames)], Cyan, message, Reset)
		time.Sleep(80 * time.Millisecond)
		i++
	}
	fmt.Printf("\r%s  ✓ %s%s\n", BrightGreen, message, Reset)
}

// PrintSystemInfo печатает системную информацию
func (ui *UI) PrintSystemInfo() {
	ui.PrintSection("Системная информация")
	ui.PrintKeyValue("OS:", runtime.GOOS)
	ui.PrintKeyValue("Arch:", runtime.GOARCH)
	ui.PrintKeyValue("Go:", runtime.Version())
	ui.PrintKeyValue("CPUs:", fmt.Sprintf("%d", runtime.NumCPU()))
	ui.PrintKeyValue("Goroutines:", fmt.Sprintf("%d", runtime.NumGoroutine()))
	ui.PrintKeyValue("SOVA Version:", Version)
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
	fmt.Printf("%s  %s%s\n", Dim+Purple, strings.Repeat("─", 50), Reset)
}