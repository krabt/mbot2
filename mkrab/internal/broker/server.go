package broker

import (
	"fmt"
	"log/slog"

	"mkrab/internal/config"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

// Server 组装 MQTT 服务及其认证、连接审计和消息持久化组件。
type Server struct {
	server *mqtt.Server
}

func NewServer(cfg *config.Config, listenerSet ListenerSet, auth *AuthACLHook, connections *ClientConnectionHook, store *MessageStore) (*Server, error) {
	server := mqtt.New(&mqtt.Options{InlineClient: true})
	if err := server.AddHook(auth, nil); err != nil {
		return nil, fmt.Errorf("add authentication hook: %w", err)
	}
	if err := server.AddHook(connections, nil); err != nil {
		return nil, fmt.Errorf("add connection hook: %w", err)
	}
	for _, listener := range listenerSet {
		if err := server.AddListener(listener); err != nil {
			return nil, fmt.Errorf("add listener %q: %w", listener.ID(), err)
		}
	}
	if *cfg.MessageStore.Enabled {
		if err := server.AddHook(store, nil); err != nil {
			return nil, fmt.Errorf("add message store hook: %w", err)
		}
		slog.Info("message store enabled", "topic", cfg.MessageStore.Topic, "database", cfg.Database)
	}
	return &Server{server: server}, nil
}

func (s *Server) Run() error   { return s.server.Serve() }
func (s *Server) Close() error { return s.server.Close() }

func (s *Server) Publish(topic string, payload []byte, retain bool, qos byte) error {
	return s.server.Publish(topic, payload, retain, qos)
}

func (s *Server) Subscribe(filter string, id int, handler func(*mqtt.Client, packets.Subscription, packets.Packet)) error {
	return s.server.Subscribe(filter, id, handler)
}

func (s *Server) Unsubscribe(filter string, id int) error {
	return s.server.Unsubscribe(filter, id)
}
