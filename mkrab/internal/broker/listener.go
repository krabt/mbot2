package broker

import (
	"crypto/tls"
	"fmt"

	"mkrab/internal/config"

	"github.com/mochi-mqtt/server/v2/listeners"
)

type ListenerSet []listeners.Listener

// NewListeners 根据配置创建已启用的四类 MQTT Listener。
func NewListeners(cfg *config.Config) (ListenerSet, error) {
	result := make(ListenerSet, 0, 4)
	if cfg.Listeners.TCP.IsEnabled() {
		result = append(result, listeners.NewTCP(listeners.Config{ID: "tcp", Address: cfg.Listeners.TCP.Address}))
	}
	if cfg.Listeners.TLS.IsEnabled() {
		tlsConfig, err := loadTLSConfig(cfg.Listeners.TLS)
		if err != nil {
			return nil, fmt.Errorf("load MQTT TLS certificate: %w", err)
		}
		result = append(result, listeners.NewTCP(listeners.Config{ID: "tls", Address: cfg.Listeners.TLS.Address, TLSConfig: tlsConfig}))
	}
	if cfg.Listeners.WebSocket.IsEnabled() {
		result = append(result, listeners.NewWebsocket(listeners.Config{ID: "websocket", Address: cfg.Listeners.WebSocket.Address}))
	}
	if cfg.Listeners.WSS.IsEnabled() {
		tlsConfig, err := loadTLSConfig(cfg.Listeners.WSS)
		if err != nil {
			return nil, fmt.Errorf("load MQTT WSS certificate: %w", err)
		}
		result = append(result, listeners.NewWebsocket(listeners.Config{ID: "wss", Address: cfg.Listeners.WSS.Address, TLSConfig: tlsConfig}))
	}
	return result, nil
}

func loadTLSConfig(cfg config.ListenerConfig) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
	}, nil
}
