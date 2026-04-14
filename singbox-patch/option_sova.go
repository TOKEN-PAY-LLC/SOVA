//go:build ignore

package sova_patch

// SOVAOutboundOptions — конфигурация SOVA outbound
type SOVAOutboundOptions struct {
	DialerOptions

	// Адрес SOVA сервера
	ServerOptions ServerOptions `json:",inline"`

	// Pre-shared key для SOVA протокола (аутентификация клиент↔сервер)
	// По умолчанию: "sova-protocol-v1-key-2026"
	PSK string `json:"psk,omitempty"`

	// Список доменов для SNI spoofing (DPI evasion)
	// По умолчанию: ["www.google.com", "cdn.cloudflare.com", ...]
	SNIList []string `json:"sni_list,omitempty"`

	// Размер фрагмента ClientHello (байт) для обхода DPI
	// 0 = отключить, по умолчанию = 2
	FragmentSize int `json:"fragment_size,omitempty"`

	// Jitter между фрагментами (мс), по умолчанию = 25
	FragmentJitter int `json:"fragment_jitter,omitempty"`

	// TLS настройки (опционально, для WS через nginx)
	TLS *OutboundTLSOptions `json:"tls,omitempty"`
}
