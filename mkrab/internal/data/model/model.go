package model

//go:generate go run ../../../cmd/mkrab gen

import (
	"time"

	"gorm.io/gorm"
)

const (
	OperationPublish       = "publish"
	OperationSubscribe     = "subscribe"
	EffectAllow            = "allow"
	EffectDeny             = "deny"
	ConnectionConnected    = "connected"
	ConnectionDisconnected = "disconnected"
)

type User struct {
	ID           uint64 `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Username     string         `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string         `gorm:"size:255;not null"`
	Enabled      bool           `gorm:"not null;default:true"`
	Roles        []Role         `gorm:"many2many:mqtt_user_role"`
}

func (User) TableName() string { return "mqtt_user" }

type Role struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string         `gorm:"size:255;uniqueIndex;not null"`
	ACLRules  []ACLRule      `gorm:"foreignKey:RoleID"`
}

func (Role) TableName() string { return "mqtt_role" }

type ACLRule struct {
	ID          uint64 `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	RoleID      uint64         `gorm:"not null;index;uniqueIndex:idx_acl_rule"`
	Operation   string         `gorm:"size:16;not null;uniqueIndex:idx_acl_rule"`
	Effect      string         `gorm:"size:8;not null;uniqueIndex:idx_acl_rule"`
	TopicFilter string         `gorm:"size:65535;not null;uniqueIndex:idx_acl_rule"`
}

func (ACLRule) TableName() string { return "mqtt_acl_rule" }

type ReceivedMessage struct {
	ID         uint64 `gorm:"primaryKey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	ReceivedAt time.Time      `gorm:"not null;index"`
	ClientID   string         `gorm:"size:65535;index"`
	Username   string         `gorm:"size:65535;index"`
	Topic      string         `gorm:"size:65535;not null;index"`
	Payload    string         `gorm:"type:json"`
	QoS        uint8          `gorm:"not null"`
	Retained   bool           `gorm:"not null"`
	Duplicate  bool           `gorm:"not null"`
}

func (ReceivedMessage) TableName() string { return "received_message" }

// Topic is a distinct MQTT topic that has been persisted by the message store.
type Topic struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Topic     string         `gorm:"size:65535;not null;uniqueIndex"`
}

func (Topic) TableName() string { return "mqtt_topic" }

type ClientConnection struct {
	ID                 uint64 `gorm:"primaryKey"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
	ClientID           string         `gorm:"size:65535;not null;index"`
	Status             string         `gorm:"size:16;not null;default:disconnected;index"`
	Username           string         `gorm:"size:255;index"`
	RemoteAddress      string         `gorm:"size:65535"`
	Listener           string         `gorm:"size:255"`
	ProtocolVersion    uint8          `gorm:"not null"`
	CleanSession       bool           `gorm:"not null"`
	ConnectedAt        time.Time      `gorm:"not null;index"`
	DisconnectedAt     *time.Time     `gorm:"index"`
	LastDisconnectedIP string         `gorm:"size:65535"`
	DisconnectReason   string         `gorm:"size:65535"`
	SessionExpired     bool           `gorm:"not null;default:false"`
}

func (ClientConnection) TableName() string { return "mqtt_client_connection" }

// UserRole 是显式的多对多中间表，以便关联记录同样支持时间戳和软删除。
type UserRole struct {
	UserID    uint64 `gorm:"primaryKey"`
	RoleID    uint64 `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (UserRole) TableName() string { return "mqtt_user_role" }
