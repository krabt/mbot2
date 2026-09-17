package broker

import (
	"context"
	"errors"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"mkrab/internal/config"
	"mkrab/internal/data"
	"mkrab/internal/data/model"
	"testing"

	"gorm.io/gorm"
)

func TestMessageAndConnectionPersistence(t *testing.T) {
	db, err := data.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	client := &mqtt.Client{ID: "device-1", Properties: mqtt.ClientProperties{Username: []byte("alice"), ProtocolVersion: 5, Clean: true}, Net: mqtt.ClientConnection{Remote: "127.0.0.1:1234", Listener: "tcp1"}}
	store := NewMessageStore(&config.Config{MessageStore: config.MessageStoreConfig{Topic: "#"}}, db.Repo.Messages, db.Repo.Topics)
	store.OnPublished(client, packets.Packet{TopicName: "$SYS/broker/uptime"})
	if _, err := db.Repo.Messages.First(context.Background()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("system topic was persisted, err=%v", err)
	}
	packet := packets.Packet{TopicName: "devices/1/status", Payload: []byte(`{"status":"online"}`), FixedHeader: packets.FixedHeader{Qos: 1, Retain: true}}
	store.OnPublished(client, packet)
	message, err := db.Repo.Messages.First(context.Background())
	if err != nil || message.Payload != string(packet.Payload) || message.ClientID != client.ID || message.Username != string(client.Properties.Username) {
		t.Fatalf("message=%+v err=%v", message, err)
	}
	store.OnPublished(client, packet)
	var topicCount int64
	if err := db.DB.Model(&model.Topic{}).Where("topic = ?", packet.TopicName).Count(&topicCount).Error; err != nil || topicCount != 1 {
		t.Fatalf("topic count=%d err=%v", topicCount, err)
	}
	hook := NewClientConnectionHook(db.Repo.Connections)
	hook.OnSessionEstablished(client, packets.Packet{})
	hook.OnDisconnect(client, errors.New("connection reset"), true)
	connection, err := db.Repo.Connections.First(context.Background())
	if err != nil || connection.Status != model.ConnectionDisconnected || connection.DisconnectedAt == nil || connection.LastDisconnectedIP != "127.0.0.1" || connection.DisconnectReason != "connection reset" {
		t.Fatalf("connection=%+v err=%v", connection, err)
	}
	connectionID := connection.ID
	previousDisconnectedAt := *connection.DisconnectedAt
	client.Net.Remote = "127.0.0.2:5678"
	hook.OnSessionEstablished(client, packets.Packet{})
	connection, err = db.Repo.Connections.First(context.Background())
	if err != nil || connection.ID != connectionID || connection.Status != model.ConnectionConnected || connection.RemoteAddress != "127.0.0.2" || connection.DisconnectedAt == nil || !connection.DisconnectedAt.Equal(previousDisconnectedAt) || connection.LastDisconnectedIP != "127.0.0.1" || connection.DisconnectReason != "connection reset" {
		t.Fatalf("connection was not updated: %+v err=%v", connection, err)
	}
	var count int64
	if err := db.DB.Model(&model.ClientConnection{}).Where("client_id = ?", client.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("client connection count=%d err=%v", count, err)
	}
}
