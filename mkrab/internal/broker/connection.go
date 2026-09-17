package broker

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"mkrab/internal/data/model"
	"mkrab/internal/data/repo"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type ClientConnectionHook struct {
	mqtt.HookBase
	repository    *repo.ConnectionRepository
	connectionIDs sync.Map
}

func NewClientConnectionHook(repository *repo.ConnectionRepository) *ClientConnectionHook {
	return &ClientConnectionHook{repository: repository}
}

func (h *ClientConnectionHook) ID() string { return "sqlite-client-connections" }
func (h *ClientConnectionHook) Provides(hook byte) bool {
	return hook == mqtt.OnSessionEstablished || hook == mqtt.OnDisconnect
}

func (h *ClientConnectionHook) OnSessionEstablished(cl *mqtt.Client, _ packets.Packet) {
	record := &model.ClientConnection{
		ClientID: cl.ID, Username: string(cl.Properties.Username), RemoteAddress: remoteHost(cl.Net.Remote),
		Listener: cl.Net.Listener, ProtocolVersion: cl.Properties.ProtocolVersion,
		Status: model.ConnectionConnected, CleanSession: cl.Properties.Clean, ConnectedAt: time.Now().UTC(),
	}
	if err := h.repository.CreateOrUpdate(context.Background(), record); err != nil {
		slog.Error("save client connection", "client", cl.ID, "error", err)
		return
	}
	h.connectionIDs.Store(cl, record.ID)
}

func (h *ClientConnectionHook) OnDisconnect(cl *mqtt.Client, disconnectErr error, expire bool) {
	id, ok := h.connectionIDs.LoadAndDelete(cl)
	if !ok {
		return
	}
	reason := ""
	if disconnectErr != nil {
		reason = disconnectErr.Error()
	}
	err := h.repository.MarkDisconnected(context.Background(), id.(uint64), time.Now().UTC(), remoteHost(cl.Net.Remote), reason, expire)
	if err != nil {
		slog.Error("update client disconnect", "client", cl.ID, "error", err)
	}
}

func remoteHost(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		return host
	}
	return strings.Trim(address, "[]")
}
