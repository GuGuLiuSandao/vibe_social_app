package websocket

import (
	"social_app/internal/db"
	"social_app/internal/logger"
	"social_app/internal/redis"
	"social_app/internal/service"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Max protobuf frame size. Needed for avatar upload over WS.
	maxWSMessageSize = 8 * 1024 * 1024
)

type Client struct {
	ID     uint
	Conn   *websocket.Conn
	Send   chan []byte
	Server *Server
	Host   string
	Scheme string
}

type Server struct {
	Clients         map[uint]*Client
	Broadcast       chan []byte
	Register        chan *Client
	Unregister      chan *Client
	Mutex           sync.RWMutex
	chatService     *service.ChatService
	userService     *service.UserService
	relationService *service.RelationService
	topicRooms      map[string]*topicRoomState
	topicUserRoom   map[uint]string
	topicMessageSeq uint64
}

func NewServer() *Server {
	database := db.GetDB()
	relationService := service.NewRelationService(database)
	server := &Server{
		Clients:         make(map[uint]*Client),
		Broadcast:       make(chan []byte),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		chatService:     service.NewChatService(relationService),
		userService:     service.NewUserService(),
		relationService: relationService,
		topicRooms:      make(map[string]*topicRoomState),
		topicUserRoom:   make(map[uint]string),
	}
	server.initTopicRooms()
	return server
}

func (s *Server) Run() {
	for {
		select {
		case client := <-s.Register:
			replacedClient, total := s.registerClient(client)
			if replacedClient != nil && replacedClient.Conn != nil {
				replacedClient.Conn.Close()
			}

			if err := redis.SetUserOnline(client.ID); err != nil {
				logger.Error("Failed to set user %d online: %v", client.ID, err)
			}
			logger.Info("User %d connected, total online: %d", client.ID, total)

		case client := <-s.Unregister:
			s.disconnectClient(client)

		case message := <-s.Broadcast:
			clients := s.clientSnapshot()
			logger.Debug("Broadcasting message to %d clients", len(clients))
			for _, client := range clients {
				select {
				case client.Send <- message:
				default:
					s.disconnectClient(client)
				}
			}
		}
	}
}

func (s *Server) registerClient(client *Client) (*Client, int) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	replacedClient := s.Clients[client.ID]
	s.Clients[client.ID] = client
	if replacedClient == client {
		replacedClient = nil
	}
	return replacedClient, len(s.Clients)
}

func (s *Server) unregisterClient(client *Client) (bool, int) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	currentClient, ok := s.Clients[client.ID]
	if !ok || currentClient != client {
		return false, len(s.Clients)
	}

	delete(s.Clients, client.ID)
	return true, len(s.Clients)
}

func (s *Server) clientSnapshot() []*Client {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	clients := make([]*Client, 0, len(s.Clients))
	for _, client := range s.Clients {
		clients = append(clients, client)
	}
	return clients
}

func (s *Server) disconnectClient(client *Client) bool {
	removed, total := s.unregisterClient(client)
	if !removed {
		return false
	}

	if client.Conn != nil {
		client.Conn.Close()
	}
	if _, err := s.LeaveTopicRoom(client.ID, ""); err != nil {
		logger.Error("Failed to cleanup topic room for user %d: %v", client.ID, err)
	}
	if err := redis.SetUserOffline(client.ID); err != nil {
		logger.Error("Failed to set user %d offline: %v", client.ID, err)
	}
	logger.Info("User %d disconnected, total online: %d", client.ID, total)
	return true
}

func (s *Server) SendToUser(userID uint, message []byte) {
	s.Mutex.RLock()
	client, ok := s.Clients[userID]
	s.Mutex.RUnlock()

	if ok {
		logger.Debug("Sending message to user %d", userID)
		select {
		case client.Send <- message:
		default:
			s.Unregister <- client
		}
	} else {
		logger.Debug("User %d is not online, cannot send message", userID)
	}
}

func (s *Server) SendToConversation(conversationID uint, message []byte) {
	subscribers, err := redis.GetConversationSubscribers(conversationID)
	if err != nil {
		logger.Error("Failed to get conversation subscribers: %v", err)
		return
	}

	logger.Debug("Sending message to conversation %d, %d subscribers", conversationID, len(subscribers))
	for _, userID := range subscribers {
		s.SendToUser(userID, message)
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Server.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxWSMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	// Handle Ping from client (e.g. browser or other clients that send Pings)
	c.Conn.SetPingHandler(func(appData string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		err := c.Conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(writeWait))
		if err == websocket.ErrCloseSent {
			return nil
		} else if e, ok := err.(interface{ Temporary() bool }); ok && e.Temporary() {
			return nil
		}
		return err
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Debug("WebSocket unexpected close error for user %d: %v", c.ID, err)
			} else {
				logger.Debug("WebSocket read error for user %d: %v", c.ID, err)
			}
			break
		}
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		c.Server.HandleMessage(c, message)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.Conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				logger.Error("WebSocket write error for user %d: %v", c.ID, err)
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
