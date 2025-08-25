# 🚀 Real-time WebSocket Todo API

A high-performance, scalable WebSocket implementation for real-time Todo management with Focus Session Mode and collaborative features.

## ✨ Features

### 🔗 Real-time WebSocket Communication

- **Bidirectional Updates**: Instant create, update, delete, and complete events
- **Thousands of Concurrent Users**: Optimized for high-performance scaling
- **User-scoped Rooms**: Dynamic routing with efficient user_id-based isolation
- **Cross-device Synchronization**: All connected sessions receive real-time updates

### 🎯 Focus Session Mode

- **Single-task Focus**: Enter focus mode on specific todos
- **Cross-device Notifications**: Active focus status visible across all sessions
- **Time Tracking**: Automatic time tracking with periodic updates
- **Collaborative Awareness**: See when others are focusing on tasks
- **Session Management**: Automatic cleanup and recovery

### 💾 Database Integration

- **Real-time Sync**: Seamless integration with existing Todo services
- **Event Broadcasting**: All CRUD operations emit WebSocket events
- **Data Consistency**: Maintains data integrity across real-time updates
- **Efficient Caching**: Redis-based caching for optimal performance

### 📈 Scalable Architecture

- **Horizontal Scaling**: Redis pub/sub for multi-server deployments
- **Session Management**: Sticky sessions with Redis storage
- **Event-driven Design**: Modular, extensible event system
- **Production Ready**: Robust error handling and monitoring

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client App    │    │   Client App    │    │   Client App    │
│   (Browser)     │    │   (Mobile)      │    │   (Desktop)     │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────▼─────────────┐
                    │    WebSocket Server      │
                    │                          │
                    │  ┌─────────────────────┐ │
                    │  │    WebSocket Hub    │ │
                    │  │   (User Rooms)      │ │
                    │  └─────────────────────┘ │
                    │                          │
                    │  ┌─────────────────────┐ │
                    │  │  Focus Manager      │ │
                    │  │  (Session Tracking) │ │
                    │  └─────────────────────┘ │
                    │                          │
                    │  ┌─────────────────────┐ │
                    │  │  Event Broadcaster  │ │
                    │  │  (Real-time Events) │ │
                    │  └─────────────────────┘ │
                    └────────────┬─────────────┘
                                 │
    ┌──────────────┬─────────────┼─────────────┬──────────────┐
    │              │             │             │              │
┌───▼───┐     ┌────▼────┐   ┌────▼────┐   ┌────▼────┐    ┌───▼───┐
│ Redis │     │Database │   │Todo API │   │Auth API │    │ Queue │
│Cache  │     │(PostgreSQL) │         │   │         │    │(Optional)
└───────┘     └─────────┘   └─────────┘   └─────────┘    └───────┘
```

## 🚀 Quick Start

### 1. Server Setup

The WebSocket server is automatically initialized in `main.go`:

```go
// Initialize WebSocket hub
wsHub := websocket.NewHub(redisClient)
ctx := context.Background()
go wsHub.Run(ctx)

// Initialize WebSocket-enabled services
wsHandler := handler.NewWebSocketHandler(wsHub, jwtManager)
wsBroadcaster := wsHandler.GetBroadcaster()

wsTodoService := service.NewWebSocketTodoService(todoService, wsBroadcaster)
```

### 2. Client Connection

#### JavaScript Client

```javascript
import { WebSocketTodoClient } from "./websocket-client.js";

const client = new WebSocketTodoClient({
  baseUrl: "ws://localhost:8080",
  token: "your-jwt-token",
});

// Connect to server
await client.connect();
```

#### HTML/Browser

```html
<!DOCTYPE html>
<html>
  <head>
    <title>WebSocket Todo App</title>
  </head>
  <body>
    <script src="websocket-client.js"></script>
    <script>
      const client = new WebSocketTodoClient({
        baseUrl: "ws://localhost:8080",
        token: localStorage.getItem("jwt_token"),
      });

      client.connect();
    </script>
  </body>
</html>
```

### 3. Basic Usage

#### Real-time Todo Operations

```javascript
// Create todo with real-time broadcast
const todo = await client.createTodo({
  task_name: "Important Task",
  project_id: "project-uuid",
});

// Listen for real-time updates
client.on("todo_created", (data) => {
  console.log("New todo:", data.todo);
  updateUI(data.todo);
});

client.on("todo_updated", (data) => {
  console.log("Todo updated:", data.todo);
  console.log("Changes:", data.changes);
  updateUI(data.todo);
});
```

#### Focus Session Management

```javascript
// Start focus session
await client.startFocusSession("todo-uuid", "Important Task", "focus");

// Listen for focus events
client.on("focus_session_started", (data) => {
  showFocusMode(data.session);
  startFocusTimer(data.session.start_time);
});

client.on("focus_timer_update", (data) => {
  updateFocusTimer(data.session.duration);
});

// End focus session
await client.endFocusSession();
```

## 📡 WebSocket Endpoints

### Connection

- `GET /api/v1/ws/connect?token={JWT}` - WebSocket upgrade endpoint

### Management (HTTP)

- `GET /api/v1/ws/users` - Get connected users
- `GET /api/v1/ws/sessions` - Get user sessions
- `GET /api/v1/ws/status` - System status
- `GET /api/v1/ws/focus` - Current focus session
- `POST /api/v1/ws/focus/start` - Start focus session
- `POST /api/v1/ws/focus/end` - End focus session

### Real-time API (HTTP + WebSocket Events)

- `POST /api/v1/ws/api/todos` - Create todo + broadcast
- `PUT /api/v1/ws/api/todos/:id` - Update todo + broadcast
- `DELETE /api/v1/ws/api/todos/:id` - Delete todo + broadcast
- `POST /api/v1/ws/api/todos/:id/complete` - Complete todo + broadcast

## 🎯 Focus Session Features

### Session Types

- **Focus**: Regular focused work session
- **Pomodoro**: 25-minute focused work session
- **Break**: Short break session

### Key Features

- ⏱️ **Automatic Time Tracking**: Precise time measurement
- 🔄 **Real-time Updates**: Timer updates every minute
- 📱 **Cross-device Sync**: Focus status visible on all devices
- 🚫 **Single Session**: Only one active focus session per user
- 🔄 **Session Recovery**: Automatic recovery after reconnection
- 📊 **Analytics**: Focus session statistics and patterns

### Usage Example

```javascript
// Start 25-minute Pomodoro session
await client.startFocusSession("todo-uuid", "Write Documentation", "pomodoro");

// Switch focus to different task
await client.switchFocus("other-todo-uuid", "Code Review");

// Get current focus status
const status = await client.getFocusStatus();
if (status.active_session) {
  console.log("Currently focusing on:", status.active_session.todo_title);
}
```

## 📊 Real-time Events

### Todo Events

```javascript
// Listen for all todo events
client.on("todo_created", (data) => {
  /* Handle new todo */
});
client.on("todo_updated", (data) => {
  /* Handle todo update */
});
client.on("todo_deleted", (data) => {
  /* Handle todo deletion */
});
client.on("todo_completed", (data) => {
  /* Handle completion */
});
client.on("todo_pinned", (data) => {
  /* Handle pin/unpin */
});
```

### Project Events

```javascript
client.on("project_created", (data) => {
  /* New project */
});
client.on("project_updated", (data) => {
  /* Project changes */
});
client.on("project_deleted", (data) => {
  /* Project removed */
});
```

### User Activity

```javascript
client.on("user_activity", (data) => {
  if (data.activity === "user_typing") {
    showTypingIndicator(data.user_id, data.todo_id);
  }
});

// Send typing notifications
client.sendUserTyping("todo-uuid");
client.sendUserStoppedTyping("todo-uuid");
```

## 🔧 Configuration

### Environment Variables

```bash
# Redis Configuration (for scaling)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# WebSocket Configuration
WS_MAX_CONNECTIONS=1000
WS_MESSAGE_SIZE_LIMIT=1048576  # 1MB
WS_PING_INTERVAL=54s
WS_PONG_TIMEOUT=60s

# Focus Session Configuration
FOCUS_TIMER_INTERVAL=60s  # Timer update frequency
FOCUS_SESSION_TIMEOUT=2h  # Auto-cleanup timeout
```

### Server Configuration

```go
// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // Configure allowed origins in production
        return true
    },
}
```

## 🔐 Security

### Authentication

- **JWT Token Required**: All connections require valid JWT
- **Token Validation**: Tokens validated on connection and periodically
- **Session Security**: Unique session IDs prevent conflicts
- **Origin Checking**: Configurable origin validation

### Rate Limiting

- **Connection Limits**: Per-IP connection limits
- **Message Frequency**: Per-connection message rate limits
- **API Integration**: Respects existing HTTP API rate limits

### Data Validation

- **Message Validation**: All incoming messages validated
- **Permission Checks**: User permissions enforced
- **Error Handling**: Secure error responses

## 📈 Performance & Scaling

### Single Server Performance

- **Concurrent Connections**: 1000+ concurrent users
- **Message Throughput**: High-frequency real-time updates
- **Memory Efficiency**: Optimized connection management
- **CPU Usage**: Event-driven, non-blocking architecture

### Horizontal Scaling

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ WS Server 1 │    │ WS Server 2 │    │ WS Server 3 │
└──────┬──────┘    └──────┬──────┘    └──────┬──────┘
       │                  │                  │
       └──────────────────┼──────────────────┘
                          │
                    ┌─────▼─────┐
                    │   Redis   │
                    │  Pub/Sub  │
                    └───────────┘
```

#### Redis Pub/Sub Events

- **Cross-server Sync**: Events propagated across all server instances
- **Session State**: Shared session state via Redis
- **Focus Sessions**: Global focus session tracking
- **Load Balancing**: Supports multiple server instances

### Monitoring

```javascript
// Get system status
const status = await client.getSystemStatus();
console.log("Active connections:", status.active_connections);
console.log("Active users:", status.active_users);
console.log("Focus sessions:", status.active_focus_sessions);
```

## 🛠️ Development

### Local Development

```bash
# Start the server
go run cmd/server/main.go

# Open the example client
open examples/websocket-client.html

# Or use the JavaScript client
node examples/websocket-test.js
```

### Testing WebSocket Connection

```bash
# Using wscat
npm install -g wscat
wscat -c "ws://localhost:8080/api/v1/ws/connect?token=YOUR_JWT_TOKEN"

# Send test message
{"type": "ping", "payload": {"message": "test"}}
```

### Debug Logging

```go
// Enable WebSocket debug logging
log.SetLevel(log.DebugLevel)
```

## 🔍 Troubleshooting

### Common Issues

#### Connection Failed

```javascript
// Check token validity
if (!token || token.split(".").length !== 3) {
  console.error("Invalid JWT token format");
}

// Check server status
fetch("/api/v1/health")
  .then((response) => response.json())
  .then((data) => console.log("Server status:", data));
```

#### Focus Session Issues

```javascript
// Check for existing session
client.getFocusStatus().then((status) => {
  if (status.active_session) {
    console.log("Already have active session:", status.active_session);
    // End existing session first
    client.endFocusSession();
  }
});
```

#### Event Not Received

```javascript
// Check connection status
console.log("Connected:", client.isConnected);

// Verify event handlers
client.on("todo_created", (data) => {
  console.log("Todo created event received:", data);
});

// Check session ID conflicts
console.log("Session ID:", client.sessionId);
```

### Debug Events

```javascript
// Log all incoming messages
client.on("message_received", (message) => {
  console.log("Received:", message);
});

// Log all outgoing messages
client.on("message_sent", (data) => {
  console.log("Sent:", data);
});
```

## 📚 Examples

See the `examples/` directory for complete working examples:

- **[websocket-client.html](../examples/websocket-client.html)**: Interactive browser demo
- **[websocket-client.js](../examples/websocket-client.js)**: JavaScript client library
- **[WEBSOCKET_API_DOCUMENTATION.md](./WEBSOCKET_API_DOCUMENTATION.md)**: Complete API reference

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

### Development Guidelines

- Follow Go best practices
- Add comprehensive tests
- Update documentation
- Use meaningful commit messages
- Test WebSocket functionality thoroughly

## 📄 License

This WebSocket implementation is part of the Todo API project and follows the same license terms.

---

**Built with ❤️ for real-time collaboration and productivity**
