package plugin

import (
	"context"
	"net"
	"sova/common"
)

// XraySOVAPlugin представляет плагин SOVA для Xray
type XraySOVAPlugin struct {
	Config *common.JSONConfig
	AI     *common.AIAdapter
}

// NewXraySOVAPlugin создает новый плагин
func NewXraySOVAPlugin(config *common.JSONConfig) *XraySOVAPlugin {
	return &XraySOVAPlugin{
		Config: config,
		AI:     common.NewAIAdapter(),
	}
}

// HandleConnection обрабатывает соединение через Xray
func (p *XraySOVAPlugin) HandleConnection(ctx context.Context, conn net.Conn, target net.Addr) error {
	// Интегрировать SOVA логику в Xray поток
	defer conn.Close()

	// Выбрать транспорт на основе AI
	transportConfig := &common.TransportConfig{
		Mode:       common.WebMirrorMode, // Или адаптивно
		ServerAddr: target.String(),
		SNI:        p.Config.SNIList[0],
	}

	sovaConn, err := common.DialTransport(transportConfig)
	if err != nil {
		return err
	}
	defer sovaConn.Conn.Close()

	// Аутентификация
	cred := &common.UserCredentials{UserID: "xray_user", Password: "xray_pass"}
	challengeBuf := make([]byte, 32)
	n, err := sovaConn.Conn.Read(challengeBuf)
	if err != nil {
		return err
	}
	challenge := &common.ZKPChallenge{Nonce: challengeBuf[:n]}

	proof, err := cred.ProvePassword(challenge, []byte(p.Config.ServerPubKey))
	if err != nil {
		return err
	}

	_, err = sovaConn.Conn.Write(proof.Response)
	if err != nil {
		return err
	}

	// Туннелирование
	tunnel := &common.TunnelReaderWriter{
		LocalConn:  conn,
		RemoteConn: sovaConn.Conn,
	}
	tunnel.StartTunnel()

	return nil
}

// SingBoxSOVAPlugin представляет плагин для Sing-Box
type SingBoxSOVAPlugin struct {
	XraySOVAPlugin
}

// NewSingBoxSOVAPlugin создает плагин для Sing-Box
func NewSingBoxSOVAPlugin(config *common.JSONConfig) *SingBoxSOVAPlugin {
	return &SingBoxSOVAPlugin{
		XraySOVAPlugin: *NewXraySOVAPlugin(config),
	}
}

// V2RaySOVAPlugin для совместимости с V2Ray
type V2RaySOVAPlugin struct {
	XraySOVAPlugin
}

// NewV2RaySOVAPlugin создает плагин для V2Ray
func NewV2RaySOVAPlugin(config *common.JSONConfig) *V2RaySOVAPlugin {
	return &V2RaySOVAPlugin{
		XraySOVAPlugin: *NewXraySOVAPlugin(config),
	}
}

// ExportToXrayConfig экспортирует конфиг для Xray
func (p *XraySOVAPlugin) ExportToXrayConfig() string {
	return `{
  "inbounds": [{
    "port": 1080,
    "protocol": "socks",
    "settings": {
      "auth": "noauth"
    },
    "sniffing": {
      "enabled": true,
      "destOverride": ["http", "tls"]
    }
  }],
  "outbounds": [{
    "protocol": "freedom"
  }, {
    "protocol": "sova",
    "settings": {
      "server": "your-server.com",
      "server_port": 443,
      "password": "your-password"
    }
  }]
}`
}

// ExportToSingBoxConfig экспортирует для Sing-Box
func (p *SingBoxSOVAPlugin) ExportToSingBoxConfig() string {
	return `{
  "inbounds": [{
    "type": "socks",
    "tag": "socks-in",
    "listen": "::",
    "listen_port": 1080
  }],
  "outbounds": [{
    "type": "sova",
    "tag": "sova-out",
    "server": "your-server.com",
    "server_port": 443,
    "password": "your-password"
  }]
}`
}