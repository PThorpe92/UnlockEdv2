import { WebSocketMessage, WebSocketEventType } from '@/common';
import { useEffect, useRef, useState } from 'react';


interface SessionMessage {
    type: 'SESSION_REQUEST' | 'SESSION_DATA';
    session_id?: string;
}

const sessionChannel = new BroadcastChannel('session-storage');

/**
 * ALMOST, JUST FIT AND FINISH LEFT
 */
export default function useWebSocketTracker(
    eventType: WebSocketEventType,
    userId: number,
    contentId?: string,
    onActivityUpdate?: (activityId: number) => void
) {
    const [activityID, setActivityID] = useState<number>(0);
    const [isConnected, setIsConnected] = useState<boolean>(false);
    const socketReference = useRef<WebSocket | null>(null);
    const activityIDReference = useRef<number>(0);

    function getSessionId() {
        let sessionId = sessionStorage.getItem('session_id');
        if (!sessionId) {
            sessionId = crypto.randomUUID();
            sessionStorage.setItem('session_id', sessionId);
            sessionChannel.postMessage({
                type: 'SESSION_DATA',
                session_id: sessionId
            });
        }
        return sessionId;
    }

    const createWebsocketConnection = () => {
        if (socketReference.current) {
            return;//websocket already exists, skipping the create sections
        }
        const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
        const host = window.location.hostname;
        const webSocketUrl = `${protocol}${host}/api/ws/listen/${eventType}`;
        //simply create the connection here
        const socket = new WebSocket(webSocketUrl);
        socketReference.current = socket;
        socket.onopen = () => {
            console.log("REMOVE ME:  websocket connected with session id:", getSessionId());
            setIsConnected(true);
            if (eventType === WebSocketEventType.SessionEvent) {
                const message: WebSocketMessage = {
                    event_type: eventType,
                    user_id: userId,
                    activity_id: activityIDReference.current,
                    session_id: getSessionId()
                };
                socket.send(JSON.stringify(message));
                console.log("REMOVE ME:  sent session tracking event:", message);
            }
        };
        //onclose handler for removing reference to websocket
        socket.onclose = (event) => {
            console.warn("REMOVE ME: websocket closed:", event.reason);
            socketReference.current = null;
            setIsConnected(false);
        };
        //onmessage handler receives messages from server, basically handles activity id for content
        socket.onmessage = (event: MessageEvent<string>) => {
            try {
                const eventData = JSON.parse(event.data) as Partial<WebSocketMessage>;
                console.log("REMOVE ME: received websocket message:", eventData.activity_id);
                if (eventData.activity_id !== undefined) {
                    if (activityIDReference.current !== 0) {
                        const visitEndMsg: WebSocketMessage = {
                            event_type: eventType,
                            user_id: userId,
                            activity_id: activityIDReference.current
                        };
                        socket.send(JSON.stringify(visitEndMsg));
                        console.log("REMOVE ME: sent visit message:", visitEndMsg);
                    }
                    setActivityID(eventData.activity_id);
                    activityIDReference.current = eventData.activity_id;
                    if (onActivityUpdate) {//just passing this back to callback function for now...
                        onActivityUpdate(eventData.activity_id);
                    }
                }
            } catch (error) {
                console.error("Error parsing webSocket message:", error);
            }
        };
    };
    const tearDownWebsocket = () => {
        if (socketReference.current) {
            console.log("REMOVE ME: closing webSocket...");
            try {
                if (eventType === WebSocketEventType.SessionEvent) {
                    const sessionMessage: WebSocketMessage = {
                        event_type: eventType,
                        user_id: userId,
                        activity_id: activityIDReference.current,
                        session_id: getSessionId(),
                        is_closing: true
                    }; 
                    socketReference.current.send(JSON.stringify(sessionMessage));
                    console.log("REMOVE ME: Sent closing message:", sessionMessage);
                }
            } catch (error) {
                console.warn("Error sending close event:", error);
            }
            socketReference.current.close();
            socketReference.current = null;
        }
    };
    useEffect(() => {
        //for broadcasting messages to other tabs/windows on same browser
        sessionChannel.onmessage = (event: MessageEvent) => {
            const data = event.data as SessionMessage;
            if (data.type === 'SESSION_REQUEST') {
                sessionChannel.postMessage({
                    type: 'SESSION_DATA',
                    session_id: getSessionId()
                });
            } else if (data.type === 'SESSION_DATA' && data.session_id) {
                sessionStorage.setItem('session_id', data.session_id);
            }
        };

        sessionChannel.postMessage({ type: 'SESSION_REQUEST' });
        createWebsocketConnection();

        const handleVisibilityChange = () => {
            if (!document.hidden) {
                console.log("REMOVE ME: Tab is now visible, checking WebSocket connection...");
                if (!socketReference.current) {
                    createWebsocketConnection();
                }
            }
        };
        const handleLogout = () => {
            console.log("REMOVE ME: Logout event detected. Closing websocket...");
            tearDownWebsocket();
        };
        const handleFocusChange = () =>{
            if (!socketReference.current) {
                console.log("REMOVE ME: Focus changed and socket didn't exist creating it again...");
                createWebsocketConnection();
            }
        };
        //add event handlers for handling closing/creating connections
        window.addEventListener('focus', handleFocusChange);
        window.addEventListener('visibilitychange', handleVisibilityChange);
        window.addEventListener('logoutEvent', handleLogout);
        return () => {
            console.log("tearing down resources");
            tearDownWebsocket();
            window.removeEventListener('focus', handleFocusChange);
            window.removeEventListener('visibilitychange', handleVisibilityChange);
            window.removeEventListener('logoutEvent', handleLogout);
        };
    }, [contentId]);

    useEffect(() => {
    //we can remove this...was just logging stuff here..TEST TEST FIRST
        if (isConnected && socketReference.current && socketReference.current.readyState === WebSocket.OPEN) {
            console.log("updated activityIDRef before sending message:", activityID);
        }
    }, [activityID]);

    return { activityID, isConnected };
}
