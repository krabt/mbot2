package broker

import (
	"context"
	"log/slog"

	"mkrab/internal/data/model"
	"mkrab/internal/data/repo"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"golang.org/x/crypto/bcrypt"
)

type AuthACLHook struct {
	mqtt.HookBase
	users *repo.UserRepository
}

func NewAuthACLHook(users *repo.UserRepository) *AuthACLHook { return &AuthACLHook{users: users} }
func (h *AuthACLHook) ID() string                            { return "sqlite-auth-acl" }
func (h *AuthACLHook) Provides(hook byte) bool {
	return hook == mqtt.OnConnectAuthenticate || hook == mqtt.OnACLCheck
}

func (h *AuthACLHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	username := string(pk.Connect.Username)
	user, err := h.users.FindEnabled(context.Background(), username)
	ok := err == nil && bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), pk.Connect.Password) == nil
	slog.Info("client authentication", "client", cl.ID, "user", username, "allowed", ok)
	return ok
}

func (h *AuthACLHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	operation := model.OperationSubscribe
	if write {
		operation = model.OperationPublish
	}
	user, err := h.users.FindEnabledWithRoles(context.Background(), string(cl.Properties.Username))
	if err != nil {
		slog.Error("load client ACL", "user", string(cl.Properties.Username), "error", err)
		return false
	}

	allowed := false
	for _, role := range user.Roles {
		for _, rule := range role.ACLRules {
			if rule.Operation != operation {
				continue
			}
			if rule.Effect == model.EffectDeny && topicFiltersOverlap(rule.TopicFilter, topic) {
				return false
			}
			if rule.Effect == model.EffectAllow && topicCoveredBy(rule.TopicFilter, topic) {
				allowed = true
			}
		}
	}
	return allowed
}
