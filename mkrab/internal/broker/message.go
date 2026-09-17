package broker

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"mkrab/internal/config"
	"mkrab/internal/data/model"
	"mkrab/internal/data/repo"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type MessageStore struct {
	mqtt.HookBase
	repository *repo.MessageRepository
	topics     *repo.TopicRepository
	filter     string
}

func NewMessageStore(cfg *config.Config, repository *repo.MessageRepository, topics *repo.TopicRepository) *MessageStore {
	return &MessageStore{repository: repository, topics: topics, filter: cfg.MessageStore.Topic}
}

func (s *MessageStore) ID() string              { return "sqlite-message-store" }
func (s *MessageStore) Provides(hook byte) bool { return hook == mqtt.OnPublished }

// OnPublished receives the publishing client, unlike an inline subscription callback.
func (s *MessageStore) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	if err := s.Save(cl, pk); err != nil {
		clientID := ""
		if cl != nil {
			clientID = cl.ID
		}
		slog.Error("save MQTT message", "client", clientID, "topic", pk.TopicName, "error", err)
	}
}

func (s *MessageStore) Save(cl *mqtt.Client, pk packets.Packet) error {
	if strings.HasPrefix(pk.TopicName, "$SYS") || !topicCoveredBy(s.filter, pk.TopicName) {
		return nil
	}
	if err := s.topics.CreateIgnoreConflict(context.Background(), pk.TopicName); err != nil {
		return err
	}

	message := &model.ReceivedMessage{
		ReceivedAt: time.Now().UTC(), Topic: pk.TopicName, Payload: string(pk.Payload),
		QoS: pk.FixedHeader.Qos, Retained: pk.FixedHeader.Retain, Duplicate: pk.FixedHeader.Dup,
	}
	if cl != nil {
		message.ClientID = cl.ID
		message.Username = string(cl.Properties.Username)
	}
	return s.repository.Create(context.Background(), message)
}
