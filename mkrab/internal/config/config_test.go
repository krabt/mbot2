package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadListenerDefaults(t *testing.T) {
	path := writeConfig(t, "database: test.db\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Listeners.TCP.IsEnabled() || cfg.Listeners.TCP.Address != ":1883" {
		t.Fatalf("unexpected TCP defaults: %+v", cfg.Listeners.TCP)
	}
	if cfg.Listeners.TLS.IsEnabled() || cfg.Listeners.WebSocket.IsEnabled() || cfg.Listeners.WSS.IsEnabled() {
		t.Fatal("secure and websocket listeners should be disabled by default")
	}
}

func TestLoadRequiresEnabledListener(t *testing.T) {
	path := writeConfig(t, "listeners:\n  tcp: {enabled: false}\n  tls: {enabled: false}\n  websocket: {enabled: false}\n  wss: {enabled: false}\n")
	if _, err := Load(path); err == nil {
		t.Fatal("configuration without listeners accepted")
	}
}

func TestLoadRequiresTLSCertificatePair(t *testing.T) {
	path := writeConfig(t, "listeners:\n  tcp: {enabled: false}\n  tls: {enabled: true}\n")
	if _, err := Load(path); err == nil {
		t.Fatal("TLS configuration without certificate accepted")
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
