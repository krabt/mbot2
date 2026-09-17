package repo

import (
	"context"
	"errors"
	"time"

	"mkrab/internal/data/model"
	"mkrab/internal/data/query"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MessageRepository struct{ query *query.Query }

func (r *MessageRepository) Create(ctx context.Context, message *model.ReceivedMessage) error {
	return r.query.ReceivedMessage.WithContext(ctx).Create(message)
}

func (r *MessageRepository) First(ctx context.Context) (*model.ReceivedMessage, error) {
	return r.query.ReceivedMessage.WithContext(ctx).First()
}

type ConnectionRepository struct{ query *query.Query }

type TopicRepository struct{ query *query.Query }

func (r *TopicRepository) CreateIgnoreConflict(ctx context.Context, topic string) error {
	return r.query.Topic.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&model.Topic{Topic: topic})
}

// CreateOrUpdate refreshes the current connection details while preserving the
// most recent disconnection details.
func (r *ConnectionRepository) CreateOrUpdate(ctx context.Context, connection *model.ClientConnection) error {
	c := r.query.ClientConnection
	existing, err := c.WithContext(ctx).Where(c.ClientID.Eq(connection.ClientID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.WithContext(ctx).Create(connection)
	}
	if err != nil {
		return err
	}

	connection.ID = existing.ID
	_, err = c.WithContext(ctx).Where(c.ID.Eq(existing.ID)).Updates(map[string]any{
		"status": connection.Status, "username": connection.Username,
		"remote_address": connection.RemoteAddress, "listener": connection.Listener,
		"protocol_version": connection.ProtocolVersion, "clean_session": connection.CleanSession,
		"connected_at": connection.ConnectedAt,
	})
	return err
}

func (r *ConnectionRepository) MarkDisconnected(ctx context.Context, id uint64, at time.Time, remoteAddress, reason string, expired bool) error {
	c := r.query.ClientConnection
	_, err := c.WithContext(ctx).Where(c.ID.Eq(id)).Updates(map[string]any{
		"status": model.ConnectionDisconnected, "disconnected_at": at,
		"last_disconnected_ip": remoteAddress, "disconnect_reason": reason, "session_expired": expired,
	})
	return err
}

func (r *ConnectionRepository) First(ctx context.Context) (*model.ClientConnection, error) {
	return r.query.ClientConnection.WithContext(ctx).First()
}

type UserRoleRepository struct{ query *query.Query }

func (r *UserRoleRepository) First(ctx context.Context) (*model.UserRole, error) {
	return r.query.UserRole.WithContext(ctx).First()
}
