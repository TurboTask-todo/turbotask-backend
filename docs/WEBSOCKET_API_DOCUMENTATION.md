# 🚀 WebSocket API Documentation

## Overview

The WebSocket API provides real-time, bidirectional communication for the Todo List application. It enables instant updates across all connected devices, focus session management, and collaborative features.

## Table of Contents

1. [Connection](#connection)
2. [Authentication](#authentication)
3. [Event Types](#event-types)
4. [Focus Session Mode](#focus-session-mode)
5. [Real-time Todo Events](#real-time-todo-events)
6. [User Activity Tracking](#user-activity-tracking)
7. [System Management](#system-management)
8. [Error Handling](#error-handling)
9. [Client Examples](#client-examples)
10. [Scaling Architecture](#scaling-architecture)

## Connection

### WebSocket Endpoint

```
ws://localhost:8080/api/v1/ws/connect?token={JWT_TOKEN}
```

### Connection Parameters

- `token` (required): JWT authentication token
- The token can be provided either as a query parameter or in the Authorization header

### Connection Flow

1. Client establishes WebSocket connection with JWT token
2. Server validates token and creates client session
3. Server sends welcome message with session info
4. Client can start sending/receiving real-time events

## Authentication

### Token-based Authentication

All WebSocket connections require a valid JWT token for authentication. The token must be provided during the initial connection.

```javascript
const wsUrl = `ws://localhost:8080/api/v1/ws/connect?token=${encodeURIComponent(
  jwtToken
)}`;
const ws = new WebSocket(wsUrl);
```

### Session Management

Each client connection receives a unique session ID that can be used to exclude the originating client from broadcast events.

## Event Types

### Message Structure

All WebSocket messages follow this structure:

```json
{
  "type": "event_type",
  "payload": {
    // Event-specific data
  },
  "timestamp": "2024-01-01T12:00:00Z",
  "event_id": "unique-event-id",
  "user_id": "user-uuid",
  "session_id": "session-id"
}
```

### Connection Events

#### `connect`

Sent when a client successfully connects.

```json
{
  "type": "connect",
  "payload": {
    "message": "Connected to WebSocket server",
    "client_id": "client-uuid",
    "user_id": "user-uuid",
    "server_time": "2024-01-01T12:00:00Z"
  }
}
```

#### `disconnect`

Sent when a client disconnects.

#### `ping` / `pong`

Heartbeat messages for connection health.

```json
{
  "type": "ping",
  "payload": { "message": "ping" }
}
```

## Focus Session Mode

Focus sessions allow users to enter a dedicated work mode on specific todos with time tracking and cross-device notifications.

### Start Focus Session

**Client → Server:**

```json
{
  "type": "focus_session_started",
  "payload": {
    "todo_id": "todo-uuid",
    "todo_title": "Task Title",
    "session_type": "focus" // "focus", "pomodoro", "break"
  }
}
```

**Server → All User Sessions:**

```json
{
  "type": "focus_session_started",
  "payload": {
    "action": "started",
    "session": {
      "id": "session-uuid",
      "user_id": "user-uuid",
      "todo_id": "todo-uuid",
      "todo_title": "Task Title",
      "start_time": "2024-01-01T12:00:00Z",
      "duration": 0,
      "is_active": true,
      "session_type": "focus",
      "client_id": "client-uuid"
    },
    "message": "Focus session started"
  }
}
```

### End Focus Session

**Client → Server:**

```json
{
  "type": "focus_session_ended",
  "payload": {}
}
```

**Server → All User Sessions:**

```json
{
  "type": "focus_session_ended",
  "payload": {
    "action": "ended",
    "session": {
      "id": "session-uuid",
      "duration": 1800000000000, // Duration in nanoseconds
      "is_active": false
    },
    "message": "Focus session ended: manual_end"
  }
}
```

### Focus Timer Updates

Sent every minute during active focus sessions:

```json
{
  "type": "focus_timer_update",
  "payload": {
    "action": "timer_update",
    "session": {
      "id": "session-uuid",
      "duration": 900000000000, // 15 minutes in nanoseconds
      "is_active": true
    }
  }
}
```

### Switch Focus Task

**Client → Server:**

```json
{
  "type": "switch_focus",
  "payload": {
    "todo_id": "new-todo-uuid",
    "todo_title": "New Task"
  }
}
```

### Get Focus Status

**Client → Server:**

```json
{
  "type": "get_focus_status",
  "payload": {}
}
```

**Server → Client:**

```json
{
  "type": "focus_status",
  "payload": {
    "active_session": {
      "id": "session-uuid",
      "todo_id": "todo-uuid",
      "todo_title": "Task Title",
      "duration": 600000000000,
      "is_active": true
    }
  }
}
```

## Real-time Todo Events

### Todo Created

```json
{
  "type": "todo_created",
  "payload": {
    "action": "created",
    "todo": {
      "id": "todo-uuid",
      "task_name": "New Task",
      "status": "not_started",
      "priority": "medium",
      "project_id": "project-uuid",
      "user_id": "user-uuid",
      "created_at": "2024-01-01T12:00:00Z"
    },
    "project_id": "project-uuid"
  }
}
```

### Todo Updated

```json
{
  "type": "todo_updated",
  "payload": {
    "action": "updated",
    "todo": {
      "id": "todo-uuid",
      "task_name": "Updated Task",
      "status": "in_progress"
    },
    "project_id": "project-uuid",
    "changes": {
      "status": {
        "old": "not_started",
        "new": "in_progress"
      }
    }
  }
}
```

### Todo Completed

```json
{
  "type": "todo_completed",
  "payload": {
    "action": "completed",
    "todo": {
      "id": "todo-uuid",
      "status": "completed",
      "completed_at": "2024-01-01T12:00:00Z"
    },
    "project_id": "project-uuid"
  }
}
```

### Todo Deleted

```json
{
  "type": "todo_deleted",
  "payload": {
    "action": "deleted",
    "todo": {
      "id": "todo-uuid"
    },
    "project_id": "project-uuid"
  }
}
```

### Todo Pinned/Unpinned

```json
{
  "type": "todo_pinned",
  "payload": {
    "action": "pinned", // or "unpinned"
    "todo": {
      "id": "todo-uuid",
      "is_pinned": true
    },
    "project_id": "project-uuid"
  }
}
```

## Subtask Events

### Subtask Created

```json
{
  "type": "subtask_created",
  "payload": {
    "action": "created",
    "subtask": {
      "id": "subtask-uuid",
      "todo_id": "todo-uuid",
      "name": "Subtask Name",
      "status": "not_started"
    },
    "todo_id": "todo-uuid"
  }
}
```

### Subtask Updated

```json
{
  "type": "subtask_updated",
  "payload": {
    "action": "updated",
    "subtask": {
      "id": "subtask-uuid",
      "status": "completed"
    },
    "todo_id": "todo-uuid",
    "changes": {
      "status": {
        "old": "not_started",
        "new": "completed"
      }
    }
  }
}
```

### Subtask Reordered

```json
{
  "type": "subtask_reordered",
  "payload": {
    "action": "reordered",
    "subtask": {
      "todo_id": "todo-uuid",
      "subtask_ids": ["uuid1", "uuid2", "uuid3"]
    },
    "todo_id": "todo-uuid"
  }
}
```

## Project Events

### Project Created

```json
{
  "type": "project_created",
  "payload": {
    "action": "created",
    "project": {
      "id": "project-uuid",
      "title": "New Project",
      "user_id": "user-uuid"
    }
  }
}
```

### Project Updated

```json
{
  "type": "project_updated",
  "payload": {
    "action": "updated",
    "project": {
      "id": "project-uuid",
      "title": "Updated Project"
    },
    "changes": {
      "title": {
        "old": "Old Title",
        "new": "Updated Project"
      }
    }
  }
}
```

## User Activity Tracking

### User Typing

**Client → Server:**

```json
{
  "type": "user_typing",
  "payload": {
    "todo_id": "todo-uuid"
  }
}
```

### User Stopped Typing

**Client → Server:**

```json
{
  "type": "user_stopped_typing",
  "payload": {
    "todo_id": "todo-uuid"
  }
}
```

### User Activity Broadcast

**Server → All Users:**

```json
{
  "type": "user_activity",
  "payload": {
    "user_id": "user-uuid",
    "username": "john_doe",
    "activity": "user_typing",
    "todo_id": "todo-uuid",
    "typing_in": {
      "todo-uuid": true
    }
  }
}
```

## System Management

### Get System Status

**Client → Server:**

```json
{
  "type": "get_system_status",
  "payload": {}
}
```

**Server → Client:**

```json
{
  "type": "system_status",
  "payload": {
    "active_connections": 25,
    "active_users": 15,
    "active_focus_sessions": 8,
    "server_time": "2024-01-01T12:00:00Z",
    "uptime": 86400000000000 // nanoseconds
  }
}
```

### Notifications

```json
{
  "type": "notification",
  "payload": {
    "title": "Task Completed! 🎉",
    "message": "Great job completing: Important Task",
    "type": "success", // "info", "success", "warning", "error"
    "action": "view_task",
    "action_url": "/todos/todo-uuid"
  }
}
```

## Error Handling

### Error Response

```json
{
  "type": "error",
  "payload": {
    "code": "INVALID_TODO_ID",
    "message": "Invalid Todo ID format",
    "details": "Expected UUID format"
  }
}
```

### Success Response

```json
{
  "type": "success",
  "payload": {
    "message": "Focus session started successfully",
    "session": {
      "id": "session-uuid"
    }
  }
}
```

### Common Error Codes

- `INVALID_MESSAGE` - Malformed message
- `MISSING_TODO_ID` - Required todo ID not provided
- `INVALID_TODO_ID` - Invalid UUID format
- `FOCUS_SESSION_ERROR` - Focus session operation failed
- `AUTHENTICATION_REQUIRED` - Missing or invalid authentication

## Client Examples

### JavaScript/Browser

```javascript
// Initialize client
const client = new WebSocketTodoClient({
  baseUrl: "ws://localhost:8080",
  token: "your-jwt-token",
});

// Event handlers
client.on("connected", () => {
  console.log("Connected to WebSocket server");
});

client.on("todo_created", (data) => {
  console.log("New todo created:", data);
  updateTodoList(data.todo);
});

client.on("focus_session_started", (data) => {
  showFocusMode(data.session);
});

// Connect and start working
await client.connect();
await client.startFocusSession("todo-uuid", "Important Task");
```

### HTTP API Integration

All CRUD operations can be performed using HTTP endpoints that automatically broadcast WebSocket events:

```javascript
// Create todo with real-time broadcast
const result = await client.createTodo({
  task_name: "New Task",
  project_id: "project-uuid",
});

// Update todo with real-time broadcast
await client.updateTodo("todo-uuid", {
  status: "completed",
});
```

### REST API Endpoints with WebSocket Events

- `POST /api/v1/ws/api/todos` - Create todo with broadcast
- `PUT /api/v1/ws/api/todos/:id` - Update todo with broadcast
- `DELETE /api/v1/ws/api/todos/:id` - Delete todo with broadcast
- `POST /api/v1/ws/api/todos/:id/complete` - Complete todo with broadcast
- `POST /api/v1/ws/api/todos/:id/pin` - Pin/unpin todo with broadcast

## Scaling Architecture

### Redis Pub/Sub

The WebSocket system uses Redis pub/sub for horizontal scaling across multiple server instances:

```json
{
  "event_type": "focus_session_started",
  "data": {
    "user_id": "user-uuid",
    "todo_id": "todo-uuid",
    "session_id": "session-uuid"
  },
  "timestamp": "2024-01-01T12:00:00Z",
  "server_id": "websocket-server-1"
}
```

### Load Balancing

- Use sticky sessions or Redis for session storage
- WebSocket connections are distributed across server instances
- Events are synchronized via Redis pub/sub

### Performance Considerations

- Maximum 1MB message size
- Connection timeout: 60 seconds
- Ping interval: 54 seconds
- Automatic reconnection with exponential backoff
- Event batching for high-frequency updates

## Rate Limiting

WebSocket connections respect the same rate limiting as HTTP APIs:

- Connection attempts: Limited per IP
- Message frequency: Monitored per connection
- Focus session operations: Limited per user

## Security

### Authentication

- JWT token validation on every connection
- Token expiration handling
- Automatic disconnection on invalid tokens

### Data Validation

- All incoming messages are validated
- Malformed messages result in error responses
- User permissions checked for all operations

### Origin Checking

Configure allowed origins in production:

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        return origin == "https://your-domain.com"
    },
}
```

## Monitoring

### Connection Metrics

- Active connections per user
- Connection duration
- Message frequency
- Error rates

### Focus Session Analytics

- Session duration tracking
- Task completion rates
- User productivity metrics
- Focus session patterns

### System Health

- WebSocket server uptime
- Redis connectivity
- Database performance
- Memory usage

## Best Practices

### Client Implementation

1. Implement automatic reconnection with exponential backoff
2. Handle connection state properly
3. Use session IDs to avoid self-broadcast loops
4. Implement proper error handling
5. Use heartbeat/ping messages for connection health

### Server Implementation

1. Implement proper connection cleanup
2. Use Redis for cross-server communication
3. Monitor connection limits and memory usage
4. Implement graceful shutdown procedures
5. Log important events for debugging

### Focus Sessions

1. End sessions on disconnection
2. Limit one active session per user
3. Persist session data for recovery
4. Implement session timeouts
5. Track session analytics

This comprehensive WebSocket API enables real-time collaboration, focus management, and instant updates across all connected devices in your Todo application.
