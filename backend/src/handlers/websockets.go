package handlers

import (
	"UnlockEdv2/src/database"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	log "github.com/sirupsen/logrus"
)

func (srv *Server) registerWebsocketRoute() {
	srv.Mux.Handle("/api/ws/listen/{event_type}", srv.authMiddleware(srv.handleError(srv.handleWebsocketConnection)))
}

const (
	bytesBuffer = 256
)

type WebsocketEventType string

const (
	SessionEvent  WebsocketEventType = "sessions"
	VisitEvent    WebsocketEventType = "visits"
	BookmarkEvent WebsocketEventType = "bookmarks"
)

type UserActivityEvent struct {
	EventType             WebsocketEventType `json:"event_type"`
	OpenContentActivityID int64              `json:"activity_id"`
	UserID                uint               `json:"user_id"`
	SessionID             string             `json:"session_id"`
	IsClosing             bool               `json:"is_closing"`
}

func (uae *UserActivityEvent) getClientKey() string {
	return fmt.Sprintf("%s-%d", uae.EventType, uae.UserID)
}

type WsClient struct {
	Conn                  *websocket.Conn
	UserID                uint
	EventType             WebsocketEventType
	SessionID             string
	OpenContentActivityID int64
	ctx                   context.Context
	cancel                context.CancelFunc
	sendChan              chan []byte
	mutex                 sync.Mutex
}

func (ws *WsClient) getClientKey() string {
	return fmt.Sprintf("%s-%d", ws.EventType, ws.UserID)
}

type ClientManager struct {
	clients map[string]*WsClient
	mutex   sync.RWMutex
}

func newClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*WsClient),
		mutex:   sync.RWMutex{},
	}
}

func (cm *ClientManager) addClient(client *WsClient) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	clientKey := client.getClientKey()
	if cm.clients[clientKey] == nil {
		cm.clients[clientKey] = client
		log.Infof("Added client with event_type-user_id %s", clientKey)
	} else {
		log.Warnf("Client already existed with event_type-user_id %s", clientKey)
	}
}

func (cm *ClientManager) removeClient(client *WsClient, reason string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	if client.Conn == nil {
		log.Warn("Connection already closed, skipping removal.")
		return
	}
	clientKey := client.getClientKey()
	if cm.clients[clientKey] != nil {
		log.Infof("Removing client user_id %d, with key %s", client.UserID, clientKey)
		err := client.Conn.Close(websocket.StatusNormalClosure, reason)
		if err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
		delete(cm.clients, clientKey)
		client.Conn = nil //needed to explicity nil it out
		client.cancel()
	}
}

func (cm *ClientManager) notifyUser(event UserActivityEvent) {
	clientKey := event.getClientKey()
	go func() {
		for i := 0; i < 3; i++ { // retrying 3 times at most
			cm.mutex.RLock()
			client, exists := cm.clients[clientKey]
			cm.mutex.RUnlock()
			if exists {
				client.mutex.Lock()
				defer client.mutex.Unlock()
				client.send(event)
				return
			}

			log.Warnf("client not found for %s. retried (%d/3)...", clientKey, i+1)
			time.Sleep(500 * time.Millisecond) // waiting at most 1.5 sec
		}
		log.Warnf("client not found for %s after 3 retries.", clientKey)
	}()
}

func (client *WsClient) send(event UserActivityEvent) {
	log.Infof("Sending message to user_id %d, message: %d", client.UserID, event.OpenContentActivityID)
	response, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Failed to marshal event: %v", err)
		return
	}
	client.sendChan <- response
}

func (client *WsClient) writePump() {
	for {
		select {
		case message := <-client.sendChan:
			log.Infof("Writing message to user_id %d", client.UserID)
			err := client.Conn.Write(context.Background(), websocket.MessageText, message)
			if err != nil {
				log.Errorf("Failed to write message: %v", err)
				return
			}
		case <-client.ctx.Done():
			return
		}
	}
}

func (srv *Server) handleWebsocketConnection(w http.ResponseWriter, r *http.Request, log sLog) error {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return newInternalServerServiceError(err, "")
	}
	log.info("WebSocket connection established")
	user := r.Context().Value(ClaimsKey).(*Claims)
	ctx, cancel := context.WithCancel(context.Background())
	client := &WsClient{
		Conn:     conn,
		UserID:   user.UserID,
		ctx:      ctx,
		cancel:   cancel,
		sendChan: make(chan []byte, bytesBuffer),
	}
	eventStr := r.PathValue("event_type")
	validEventTypes := map[string]WebsocketEventType{
		string(SessionEvent):  SessionEvent,
		string(VisitEvent):    VisitEvent,
		string(BookmarkEvent): BookmarkEvent,
	}
	websocketEventType, ok := validEventTypes[eventStr]
	if !ok {
		return newBadRequestServiceError(errors.New("unrecognized event type"), fmt.Sprintf("event type sent was %s", eventStr))
	}
	client.EventType = websocketEventType
	//client with a connection already (other tab or window)
	srv.handleIfClientExists(client, "connected from a different device or tab")
	srv.wsClient.addClient(client)
	go client.writePump()
	go srv.handleWsHeartbeat(client)
	go srv.handleWsReader(ctx, client)
	<-ctx.Done()
	srv.wsClient.removeClient(client, "")
	return nil
}

func (cm *ClientManager) handleCleanup(db *database.DB, clientKey string) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	client, ok := cm.clients[clientKey]
	if !ok {
		return
	}
	switch client.EventType {
	case SessionEvent:
		if client.SessionID != "" {
			db.LogUserSessionEnded(client.UserID, client.SessionID)
		}
	case VisitEvent:
		if client.OpenContentActivityID > 0 {
			db.UpdateOpenContentActivityStopTS(client.OpenContentActivityID)
		}
	}
}

func (srv *Server) handleIfClientExists(client *WsClient, reason string) {
	clientKey := client.getClientKey()
	existingClient, exists := srv.wsClient.clients[clientKey]
	if exists {
		log.Warnf("client already exists for %s. closing old connection.", clientKey)
		srv.wsClient.handleCleanup(srv.Db, clientKey)
		srv.wsClient.removeClient(existingClient, reason)
	}
}

func (srv *Server) handleWsReader(ctx context.Context, client *WsClient) {
	defer client.cancel()
	for {
		_, msg, err := client.Conn.Read(ctx)
		if err != nil { //if there was an error then we should attempt a clean up
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				log.Info("WebSocket connection closed by client")
			} else {
				log.Errorf("Error reading from WebSocket: %v", err)
				srv.handleIfClientExists(client, "reading from client failed")
			}
			srv.wsClient.removeClient(client, "WebSocket read error")
			return
		}
		var event UserActivityEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			log.Warnf("Invalid message event from user %d: %v", client.UserID, err)
			continue
		}
		switch event.EventType {
		case VisitEvent:
			srv.Db.UpdateOpenContentActivityStopTS(event.OpenContentActivityID)
		case SessionEvent:
			if event.IsClosing {
				srv.Db.LogUserSessionEnded(event.UserID, event.SessionID)
			} else {
				client.SessionID = event.SessionID
				srv.Db.LogUserSessionStarted(client.UserID, event.SessionID)
			}
		}
	}
}

func (srv *Server) handleWsHeartbeat(client *WsClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-client.ctx.Done():
			return
		case <-ticker.C:
			log.Info("sending ping???")
			if err := client.Conn.Ping(client.ctx); err != nil {
				log.Errorf("Failed to send ping: %v", err)
				srv.wsClient.removeClient(client, "ping failed")
				return
			}
		}
	}
}
