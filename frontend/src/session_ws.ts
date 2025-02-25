import {
    WsMsg,
    WsEventType,
    OcActivityUpdate,
    Result,
    Success,
    ClientHello,
    WsMsgType,
    ClientGoodbye,
    Err
} from './common';

interface SessionMessage {
    type: SessionMsgType;
    session_id?: string;
}

enum SessionMsgType {
    Request = 'SESSION_REQUEST',
    Data = 'SESSION_DATA'
}
const SESSION_ID = 'session_id';

export class WebsocketSession {
    private socket: WebSocket | null = null;
    private currentActivityId = 0;
    private readonly userId: number;
    private readonly reconnectInterval: number = 5000; // 5 seconds delay before retrying
    // private readonly heartbeatInterval: number = 30000; // 30 seconds
    // private heartbeatTimer: number | undefined;
    private sessionChannel: BroadcastChannel;

    constructor(userId: number) {
        this.userId = userId;
        this.sessionChannel = new BroadcastChannel('session-storage');
        this.setupBroadcastChannel();
        this.addWindowListeners();
    }

    // sets up channel to share session IDs across tabs
    private setupBroadcastChannel(): void {
        this.sessionChannel.onmessage = (event: MessageEvent) => {
            const data = event.data as SessionMessage;
            if (data.type === SessionMsgType.Request) {
                this.sessionChannel.postMessage({
                    type: SessionMsgType.Data,
                    session_id: this.getSessionId()
                });
            } else if (data.type === SessionMsgType.Data && data.session_id) {
                sessionStorage.setItem(SESSION_ID, data.session_id);
            }
        };
        this.sessionChannel.postMessage({ type: SessionMsgType.Data });
    }

    // retrieves or generates a session id stored in sessionStorage
    private getSessionId(): string {
        let sessionId = sessionStorage.getItem(SESSION_ID);
        if (!sessionId) {
            sessionId = crypto.randomUUID();
            sessionStorage.setItem(SESSION_ID, sessionId);
            this.sessionChannel.postMessage({
                type: SessionMsgType.Data,
                session_id: sessionId
            });
        }
        return sessionId;
    }

    // adds event listeners to re-establish the connection
    private addWindowListeners(): void {
        window.addEventListener('focus', this.handleFocusChange);
        window.addEventListener(
            'visibilitychange',
            this.handleVisibilityChange
        );
        window.addEventListener('logoutEvent', this.handleLogout);
    }

    private removeWindowListeners(): void {
        window.removeEventListener('focus', this.handleFocusChange);
        window.removeEventListener(
            'visibilitychange',
            this.handleVisibilityChange
        );
        window.removeEventListener('logoutEvent', this.handleLogout);
    }

    private handleFocusChange = (): void => {
        if (!this.socket) {
            this.createConnection();
        }
    };

    private handleVisibilityChange = (): void => {
        if (!document.hidden && !this.socket) {
            this.createConnection();
        }
    };

    private handleLogout = (): void => {
        this.tearDownConnection();
    };

    // public API to connect or reconnect the websocket
    public connect(): void {
        this.createConnection();
    }

    // creates the websocket connection and sets up its event handlers
    private createConnection(): Result<void> {
        if (this.socket) return Success(); // already exists
        const webSocketUrl = `/api/ws/listen`;
        try {
            this.socket = new WebSocket(webSocketUrl);
            // eslint-disable-next-line
        } catch (error: any) {
            return Err(`failed to connect`);
        }
        this.socket.onopen = () => {
            // send an initial message with session details for client hello
            const message: WsMsg<ClientHello> = {
                event_type: WsEventType.ClientHello,
                msg: {
                    msg: 'client_hello'
                },
                user_id: this.userId,
                session_id: this.getSessionId()
            };
            this.sendMessage(message);
            // this.startHeartbeat();
        };

        this.socket.onmessage = (event: MessageEvent<string>) => {
            try {
                const data = JSON.parse(event.data) as Partial<
                    WsMsg<WsMsgType>
                >;
                // if an activity_id is received, end any prior activity and update the id.
                if (data.event_type !== undefined) {
                    if (
                        this.currentActivityId !== 0 &&
                        data.event_type === WsEventType.VisitEvent
                    ) {
                        this.notifyOpenContentActivity();
                        this.currentActivityId = (
                            data.msg as OcActivityUpdate
                        ).activity_id;
                    }
                }
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };

        this.socket.onclose = (event: CloseEvent) => {
            console.warn('WebSocket closed:', event.reason);
            this.socket = null;
            // this.stopHeartbeat();
            // Retry connection after a delay.
            setTimeout(() => this.createConnection(), this.reconnectInterval);
        };

        this.socket.onerror = (error: Event) => {
            console.error('WebSocket error:', error);
        };
        return Success();
    }

    public notifyOpenContentActivity(): void {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            const msg: WsMsg<OcActivityUpdate> = {
                event_type: WsEventType.VisitEvent,
                msg: {
                    activity_id: this.currentActivityId
                },
                user_id: this.userId,
                session_id: this.getSessionId()
            };
            this.socket.send(JSON.stringify(msg));
        } else {
            this.connect();
            this.notifyOpenContentActivity();
        }
    }

    // public API to send messages to the backend.
    public sendMessage<T>(msg: WsMsg<T>): void {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify(msg));
        } else {
            console.warn('WebSocket is not open. Message not sent:');
        }
    }

    //// starts a heartbeat
    //private startHeartbeat(): void {
    //    this.heartbeatTimer = window.setInterval(() => {
    //        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
    //            this.socket.send(JSON.stringify({event_type: WsEventType.Pong}));
    //        }
    //    }, this.heartbeatInterval);
    //}
    //
    //private stopHeartbeat(): void {
    //    if (this.heartbeatTimer) {
    //        clearInterval(this.heartbeatTimer);
    //        this.heartbeatTimer = undefined;
    //    }
    //}
    //
    // public API to tear down the connection and clean up resources.
    public tearDownConnection(): void {
        if (this.socket) {
            const sessionMessage: WsMsg<ClientGoodbye> = {
                event_type: WsEventType.ClientGoodbye,
                user_id: this.userId,
                session_id: this.getSessionId(),
                msg: { activity_id: this.currentActivityId }
            };
            this.sendMessage(sessionMessage);
            this.socket.close();
            this.socket = null;
        }
        // this.stopHeartbeat();
        this.removeWindowListeners();
        this.sessionChannel.close();
    }
}

declare global {
    interface Window {
        websocketSession?: WebsocketSession;
    }
}

export default WebsocketSession;
