package common

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	menuKeyUp    = -1
	menuKeyDown  = -2
	menuKeyEnter = -3
	menuKeyEsc   = -4
)

func menuReadKey() int {
	buf := make([]byte, 4)
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return 0
	}
	b := buf[0]
	if b == 0x1B {
		if n >= 3 && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return menuKeyUp
			case 'B':
				return menuKeyDown
			}
		}
		return menuKeyEsc
	}
	switch b {
	case '\r', '\n':
		return menuKeyEnter
	case 'k', 'K':
		return menuKeyUp
	case 'j', 'J':
		return menuKeyDown
	case 'q', 'Q':
		return menuKeyEsc
	}
	return int(b)
}

// MenuItem represents a single menu item
type MenuItem struct {
	LabelEN string
	LabelRU string
	DescEN  string
	DescRU  string
}

// Label returns localized label
func (m *MenuItem) Label() string { return T(m.LabelEN, m.LabelRU) }

// Desc returns localized description
func (m *MenuItem) Desc() string { return T(m.DescEN, m.DescRU) }

// printMenuHeader clears screen and prints a compact SOVA header
func printMenuHeader() {
	fmt.Print("\033[2J\033[H") // clear screen + move cursor home
	fmt.Printf("\r\n")
	fmt.Printf("  %s%s      {◉,◉}%s\r\n", Purple6, Bold, Reset)
	fmt.Printf("  %s%s  S O V A  Protocol v%s%s\r\n", Purple7, Bold, Version, Reset)
	fmt.Printf("  %s%s%s\r\n", Purple3, strings.Repeat("━", 42), Reset)
	fmt.Printf("\r\n")
}

// RunMenu shows an interactive arrow-key menu and returns selected index (-1 for escape)
func RunMenu(titleEN, titleRU string, items []MenuItem) int {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return 0
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0
	}
	defer term.Restore(fd, oldState)
	enableVTInput()

	selected := 0
	n := len(items)

	for {
		printMenuHeader()

		title := T(titleEN, titleRU)
		fmt.Printf("  %s%s%s%s\r\n", Purple7, Bold, title, Reset)
		fmt.Printf("  %s%s%s\r\n\r\n", Purple3, strings.Repeat("─", 50), Reset)

		for i := 0; i < n; i++ {
			lbl := items[i].Label()
			desc := items[i].Desc()
			if i == selected {
				fmt.Printf("  %s  ▸ %-26s %s%s\r\n", BgMagenta+White+Bold, lbl, desc, Reset)
			} else {
				fmt.Printf("    %s%-26s %s%s%s\r\n", Purple8, lbl, Dim, desc, Reset)
			}
		}

		fmt.Printf("\r\n")
		hint := T(
			"  ↑↓/j/k: navigate | Enter: select | Esc/q: back",
			"  ↑↓/j/k: навигация | Enter: выбор | Esc/q: назад",
		)
		fmt.Printf("  %s%s%s\r\n", Dim, hint, Reset)

		key := menuReadKey()
		switch key {
		case menuKeyUp:
			selected = (selected - 1 + n) % n
		case menuKeyDown:
			selected = (selected + 1) % n
		case menuKeyEnter:
			return selected
		case menuKeyEsc:
			return -1
		}
	}
}

// SelectLanguage shows a language picker and returns the chosen language
func SelectLanguage() Language {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return LangEN
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return LangEN
	}
	defer term.Restore(fd, oldState)
	enableVTInput()

	selected := 0

	for {
		fmt.Print("\033[2J\033[H") // clear screen
		fmt.Printf("\r\n")
		fmt.Printf("  %s%s      {◉,◉}%s\r\n", Purple6, Bold, Reset)
		fmt.Printf("  %s%s  S O V A  Protocol v%s%s\r\n", Purple7, Bold, Version, Reset)
		fmt.Printf("  %s%s%s\r\n\r\n", Purple3, strings.Repeat("━", 42), Reset)

		fmt.Printf("  %s%sSelect language / Выберите язык:%s\r\n", Purple6, Bold, Reset)
		fmt.Printf("  %s%s%s\r\n\r\n", Purple3, strings.Repeat("─", 42), Reset)

		langs := []string{"[EN]  English", "[RU]  Русский"}
		for i, l := range langs {
			if i == selected {
				fmt.Printf("  %s  ▸ %s %s\r\n", BgMagenta+White+Bold, l, Reset)
			} else {
				fmt.Printf("    %s%s%s\r\n", Purple8, l, Reset)
			}
		}
		fmt.Printf("\r\n  %s↑↓ + Enter%s\r\n", Dim, Reset)

		key := menuReadKey()
		switch key {
		case menuKeyUp, menuKeyDown:
			selected = 1 - selected
		case menuKeyEnter:
			if selected == 1 {
				return LangRU
			}
			return LangEN
		}
	}
}
