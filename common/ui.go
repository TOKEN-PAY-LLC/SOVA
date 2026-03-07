package common

import (
	"fmt"
	"os"
	"time"
)

// Owl ASCII art
const OwlIcon = `
   ,___,
   {o,o}
   /)  )
   -"  "-
  SOVA AI
`

// Colors for purple theme
const (
	Purple    = "\033[35m"
	BrightPurple = "\033[95m"
	Reset     = "\033[0m"
	Cyan      = "\033[36m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Red       = "\033[31m"
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
	fmt.Print(Purple + OwlIcon + Reset)
	fmt.Println(BrightPurple + "SOVA Protocol - Secure Obfuscated Versatile Adapter" + Reset)
	fmt.Println(Cyan + "Лучший AI-умный протокол для непробиваемой приватности" + Reset)
	fmt.Println()
}

// PrintStatus печатает статус
func (ui *UI) PrintStatus(status string, color string) {
	fmt.Printf("%s[%s] %s%s\n", color, time.Now().Format("15:04:05"), status, Reset)
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
	fmt.Printf("\r%s[%.1f%%] %s %s", Purple, percentage, bar, message + Reset)
	if current == total {
		fmt.Println()
	}
}

// PrintError печатает ошибку
func (ui *UI) PrintError(err error) {
	fmt.Printf("%s[ERROR] %v%s\n", Red, err, Reset)
}

// PrintSuccess печатает успех
func (ui *UI) PrintSuccess(message string) {
	fmt.Printf("%s[SUCCESS] %s%s\n", Green, message, Reset)
}

// PrintInfo печатает информацию
func (ui *UI) PrintInfo(message string) {
	if ui.Verbose {
		fmt.Printf("%s[INFO] %s%s\n", Cyan, message, Reset)
	}
}

// AnimateConnection анимирует подключение
func (ui *UI) AnimateConnection() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for i := 0; i < 10; i++ {
		fmt.Printf("\r%s %s Подключение к SOVA...%s", Purple+frames[i], Yellow, Reset)
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Printf("\r%s ✓ Подключено!%s\n", Green, Reset)
}

// ConfirmAction запрашивает подтверждение
func (ui *UI) ConfirmAction(prompt string) bool {
	fmt.Printf("%s%s (y/N): %s", Yellow, prompt, Reset)
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y"
}

// ExitWithError выходит с ошибкой
func (ui *UI) ExitWithError(err error) {
	ui.PrintError(err)
	os.Exit(1)
}