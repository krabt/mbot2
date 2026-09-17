package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listeners    ListenersConfig    `yaml:"listeners"`
	Database     string             `yaml:"database"`
	MessageStore MessageStoreConfig `yaml:"message_store"`
	Web          WebConfig          `yaml:"web"`
}

type ListenersConfig struct {
	TCP       ListenerConfig `yaml:"tcp"`
	TLS       ListenerConfig `yaml:"tls"`
	WebSocket ListenerConfig `yaml:"websocket"`
	WSS       ListenerConfig `yaml:"wss"`
}

type ListenerConfig struct {
	Enabled  *bool  `yaml:"enabled"`
	Address  string `yaml:"address"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

func (c ListenerConfig) IsEnabled() bool { return c.Enabled != nil && *c.Enabled }

type MessageStoreConfig struct {
	Enabled *bool  `yaml:"enabled"`
	Topic   string `yaml:"topic"`
}

type WebConfig struct {
	Enabled *bool  `yaml:"enabled"`
	Address string `yaml:"address"`
}

func (c WebConfig) IsEnabled() bool { return c.Enabled != nil && *c.Enabled }

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(b))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	applyListenerDefaults(&cfg.Listeners.TCP, true, ":1883")
	applyListenerDefaults(&cfg.Listeners.TLS, false, ":8883")
	applyListenerDefaults(&cfg.Listeners.WebSocket, false, ":8083")
	applyListenerDefaults(&cfg.Listeners.WSS, false, ":8084")
	if !cfg.Listeners.TCP.IsEnabled() && !cfg.Listeners.TLS.IsEnabled() && !cfg.Listeners.WebSocket.IsEnabled() && !cfg.Listeners.WSS.IsEnabled() {
		return nil, fmt.Errorf("at least one MQTT listener must be enabled")
	}
	for name, listener := range map[string]ListenerConfig{"tls": cfg.Listeners.TLS, "wss": cfg.Listeners.WSS} {
		if listener.IsEnabled() && (listener.CertFile == "" || listener.KeyFile == "") {
			return nil, fmt.Errorf("listeners.%s requires cert_file and key_file", name)
		}
	}
	if cfg.Database == "" {
		cfg.Database = "mqtt.db"
	}
	if cfg.MessageStore.Enabled == nil {
		enabled := true
		cfg.MessageStore.Enabled = &enabled
	}
	if cfg.MessageStore.Topic == "" {
		cfg.MessageStore.Topic = "#"
	}
	if cfg.Web.Enabled == nil {
		enabled := true
		cfg.Web.Enabled = &enabled
	}
	if cfg.Web.Address == "" {
		cfg.Web.Address = "127.0.0.1:8080"
	}
	if *cfg.MessageStore.Enabled {
		if err := validateFilter(cfg.MessageStore.Topic); err != nil {
			return nil, fmt.Errorf("message_store.topic %q: %w", cfg.MessageStore.Topic, err)
		}
	}
	return &cfg, nil
}

func applyListenerDefaults(listener *ListenerConfig, enabled bool, address string) {
	if listener.Enabled == nil {
		listener.Enabled = &enabled
	}
	if listener.Address == "" {
		listener.Address = address
	}
}

func validateFilter(filter string) error {
	if filter == "" {
		return fmt.Errorf("empty filter")
	}
	parts := strings.Split(filter, "/")
	for i, part := range parts {
		if strings.Contains(part, "#") && (part != "#" || i != len(parts)-1) {
			return fmt.Errorf("# must be the final level")
		}
		if strings.Contains(part, "+") && part != "+" {
			return fmt.Errorf("+ must occupy a level")
		}
	}
	return nil
}
