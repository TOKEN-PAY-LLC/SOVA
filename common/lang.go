package common

// Language represents a UI language
type Language int

const (
	LangEN Language = iota
	LangRU
)

// CurrentLang is the active language
var CurrentLang = LangEN

// T returns translated text based on current language
func T(en, ru string) string {
	if CurrentLang == LangRU {
		return ru
	}
	return en
}
