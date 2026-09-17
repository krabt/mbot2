package admin

import (
	"embed"
	"net/http"
	"sync"
	"sync/atomic"

	"mkrab/internal/broker"
	"mkrab/internal/data/model"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"gorm.io/gorm"
)

//go:embed index.html
var assets embed.FS

type Server struct {
	db            *gorm.DB
	broker        *broker.Server
	http          *http.Server
	subscriptions atomic.Int64
}

func New(address string, db *gorm.DB, mqtt *broker.Server) *Server {
	s := &Server{db: db, broker: mqtt}
	index, err := assets.ReadFile("index.html")
	if err != nil {
		panic(err)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/", func(c *gin.Context) { c.Data(http.StatusOK, "text/html; charset=utf-8", index) })
	api := r.Group("/api")
	api.GET("/topics", s.list(&[]model.Topic{}))
	api.GET("/messages", s.list(&[]model.ReceivedMessage{}))
	api.GET("/connections", s.list(&[]model.ClientConnection{}))
	api.GET("/users", s.list(&[]model.User{}))
	api.GET("/roles", s.list(&[]model.Role{}))
	api.GET("/acl-rules", s.list(&[]model.ACLRule{}))
	api.POST("/publish", s.publish)
	r.GET("/ws", s.ws)
	s.http = &http.Server{Addr: address, Handler: r}
	return s
}

func (s *Server) list(target any) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := s.db.Order("id desc").Limit(500).Find(target).Error; err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, target)
	}
}

func (s *Server) Run() error   { return s.http.ListenAndServe() }
func (s *Server) Close() error { return s.http.Close() }

func (s *Server) publish(c *gin.Context) {
	var request struct {
		Topic, Payload string
		QoS            byte `json:"qos"`
		Retain         bool `json:"retain"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Topic == "" {
		c.JSON(400, gin.H{"error": "topic is required"})
		return
	}
	if err := s.broker.Publish(request.Topic, []byte(request.Payload), request.Retain, request.QoS); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Status(204)
}

var upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

func (s *Server) ws(c *gin.Context) {
	filter := c.Query("topic")
	if filter == "" {
		c.String(400, "topic is required")
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	id := int(s.subscriptions.Add(1))
	var writeMu sync.Mutex
	if err := s.broker.Subscribe(filter, id, func(_ *mqtt.Client, _ packets.Subscription, pk packets.Packet) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.WriteJSON(gin.H{"topic": pk.TopicName, "payload": string(pk.Payload), "qos": pk.FixedHeader.Qos, "retain": pk.FixedHeader.Retain})
	}); err != nil {
		_ = conn.WriteJSON(gin.H{"error": err.Error()})
		_ = conn.Close()
		return
	}
	defer func() { _ = s.broker.Unsubscribe(filter, id) }()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			_ = conn.Close()
			return
		}
	}
}
