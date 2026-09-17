package broker

import (
	"context"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"golang.org/x/crypto/bcrypt"
	"mkrab/internal/data"
	"mkrab/internal/data/model"
	"testing"
)

func TestDatabaseAuthAndRoleACL(t *testing.T) {
	db, err := data.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	role := &model.Role{Name: "device"}
	if err := db.Repo.Roles.Create(ctx, role); err != nil {
		t.Fatal(err)
	}
	rules := []*model.ACLRule{{RoleID: role.ID, Operation: model.OperationPublish, Effect: model.EffectAllow, TopicFilter: "devices/#"}, {RoleID: role.ID, Operation: model.OperationPublish, Effect: model.EffectDeny, TopicFilter: "devices/private/#"}, {RoleID: role.ID, Operation: model.OperationSubscribe, Effect: model.EffectAllow, TopicFilter: "status/+"}}
	if err := db.Repo.ACLRules.Create(ctx, rules...); err != nil {
		t.Fatal(err)
	}
	user := &model.User{Username: "alice", PasswordHash: string(hash), Enabled: true}
	if err := db.Repo.Users.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	if err := db.Repo.Users.AddRole(ctx, user, role); err != nil {
		t.Fatal(err)
	}
	hook := NewAuthACLHook(db.Repo.Users)
	client := &mqtt.Client{ID: "c1", Properties: mqtt.ClientProperties{Username: []byte("alice")}}
	if !hook.OnConnectAuthenticate(client, packets.Packet{Connect: packets.ConnectParams{Username: []byte("alice"), Password: []byte("secret")}}) {
		t.Fatal("valid login rejected")
	}
	if hook.OnConnectAuthenticate(client, packets.Packet{Connect: packets.ConnectParams{Username: []byte("alice"), Password: []byte("wrong")}}) {
		t.Fatal("bad login accepted")
	}
	if !hook.OnACLCheck(client, "devices/1", true) || hook.OnACLCheck(client, "devices/private/key", true) || hook.OnACLCheck(client, "devices/#", true) {
		t.Fatal("publish ACL failed")
	}
	if !hook.OnACLCheck(client, "status/+", false) {
		t.Fatal("subscription ACL failed")
	}
}
