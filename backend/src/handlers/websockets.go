package handlers

import (
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
	//going to need a second listener here for subsssss....hmmmm...
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
	Conn      *websocket.Conn
	UserID    uint
	EventType WebsocketEventType
	ctx       context.Context
	cancel    context.CancelFunc
	sendChan  chan []byte
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

func (cm *ClientManager) addClient(clientKey string, client *WsClient) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	if cm.clients[clientKey] == nil {
		cm.clients[clientKey] = client
		log.Infof("Added client with event_type-user_id %s", clientKey)
	} else {
		log.Warnf("Client already existed with event_type-user_id %s", clientKey)
	}
}

func (cm *ClientManager) removeClient(client *WsClient, reason string) {
	client.cancel()
	err := client.Conn.Close(websocket.StatusNormalClosure, reason)
	if err != nil {
		log.Errorf("Failed to close connection: %v", err)
	}
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	if cm.clients[client.getClientKey()] != nil {
		log.Infof("Removing client user_id %d", client.UserID)
		delete(cm.clients, client.getClientKey())
	}
}

func (cm *ClientManager) notifyUser(event UserActivityEvent) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	if client, ok := cm.clients[event.getClientKey()]; ok {
		client.send(event)
	}
}

// func (cm *ClientManager) notifyUser(userId uint, message []byte) {
// 	cm.mutex.RLock()
// 	defer cm.mutex.RUnlock()
// 	if client, ok := cm.clients[userId]; ok {
// 		client.send(message)
// 	}
// }

func (client *WsClient) send(event UserActivityEvent) {
	log.Infof("Sending message to user_id %d, message: %d", client.UserID, event.OpenContentActivityID)
	response, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Failed to marshal event: %v", err)
		return
	}
	client.sendChan <- response
}

// func (client *WsClient) send(message []byte) {
// 	log.Infof("Sending message to user_id %d, message: %s", client.UserID, message)

// 	client.sendChan <- message
// }

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
	srv.wsClient.addClient(client.getClientKey(), client) //should we check to see if the user/key is there and if so close it?
	go client.writePump()
	go srv.handleWsHeartbeat(client)
	go srv.handleWsReader(ctx, client)
	<-ctx.Done()
	srv.wsClient.removeClient(client, "") //this removes client
	return nil
}

func (srv *Server) handleWsReader(ctx context.Context, client *WsClient) {
	defer client.cancel()
	for {
		_, msg, err := client.Conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				log.Info("WebSocket connection closed by client")
			} else {
				log.Errorf("Error reading from WebSocket: %v", err)
			}
			//check if this is a sessions type client. and if so log them out.
			srv.wsClient.removeClient(client, "reading from client failed")
			return
		}
		//FIXME JUST TESTING THIS HERE!!!!!!!!
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>reading websocket message, here...going on to the next step....")

		// Parse JSON event
		var event UserActivityEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			fmt.Println("error unmarsahling!!!, error is: ", err)
			log.Warnf("Invalid event from user %d: %v", client.UserID, err)
			continue
		}
		//okay so this will log
		//depending on message event type do something
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>EVENT TYPE>>>", event.EventType)
		switch event.EventType {
		case VisitEvent:
			fmt.Println("activity_id>>>>>>", event.OpenContentActivityID)
			fmt.Println("userId>>>>>>", event.UserID)
			srv.Db.UpdateOpenContentActivityStopTS(event.OpenContentActivityID)
		case SessionEvent:
			if event.IsClosing {
				srv.Db.LogUserLogout(event.UserID, event.SessionID)
			} else {
				srv.Db.LogUserLogin(client.UserID, event.SessionID)
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
