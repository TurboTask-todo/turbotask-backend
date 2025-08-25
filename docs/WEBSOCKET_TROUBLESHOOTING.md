# 🛠️ WebSocket Troubleshooting Guide

## Common Issues and Solutions

### 1. Compilation Errors

#### ✅ **Fixed: Missing Subtask Request Models**

```bash
# Error: undefined: models.CreateSubtaskRequest
# Error: undefined: models.UpdateSubtaskRequest
```

**Solution**: Added the missing request models to `internal/models/todo_models.go`:

- `CreateSubtaskRequest`
- `UpdateSubtaskRequest`

#### ✅ **Fixed: Incorrect Method Signatures**

```bash
# Error: s.SubtaskService.GetSubtaskByID undefined
# Error: too many arguments in call to s.SubtaskService.UpdateSubtask
```

**Solution**: Updated WebSocket service methods to match existing service signatures:

- Use repository methods directly where needed
- Match parameter lists with base service methods
- Include `userID` parameter in all operations

#### ✅ **Fixed: Missing WebSocket Dependencies**

```bash
# Error: could not import github.com/gorilla/websocket
```

**Solution**: Added gorilla/websocket dependency:

```bash
go get github.com/gorilla/websocket
```

### 2. Runtime Issues

#### Connection Failed

```javascript
// Symptoms: WebSocket connection fails immediately
if (!token || token.split(".").length !== 3) {
  console.error("Invalid JWT token format");
}
```

**Solutions**:

1. **Check Token Format**: Ensure JWT token has 3 parts separated by dots
2. **Verify Token Validity**: Use a fresh token from `/auth/login`
3. **Check Server Status**: Ensure server is running on port 8080

#### Authentication Errors

```javascript
// Error: 401 Unauthorized
```

**Solutions**:

1. **Get Fresh Token**:

   ```bash
   curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"user@example.com","password":"password"}'
   ```

2. **Include Token in Connection**:

   ```javascript
   const wsUrl = `ws://localhost:8080/api/v1/ws/connect?token=${encodeURIComponent(
     token
   )}`;
   ```

3. **Check Token Expiration**: JWT tokens have expiration times

#### Focus Session Issues

```javascript
// Error: Cannot start focus session
client.getFocusStatus().then((status) => {
  if (status.active_session) {
    console.log("Already have active session:", status.active_session);
    // End existing session first
    client.endFocusSession();
  }
});
```

**Solutions**:

1. **End Existing Sessions**: Only one focus session per user allowed
2. **Use Valid Todo IDs**: Ensure todo exists and user has access
3. **Check User Permissions**: Verify user owns the todo

### 3. Development Setup

#### Testing WebSocket Connection

```bash
# Using Node.js test script
node examples/websocket-test.js YOUR_JWT_TOKEN

# Using wscat
npm install -g wscat
wscat -c "ws://localhost:8080/api/v1/ws/connect?token=YOUR_JWT_TOKEN"

# Using browser demo
open examples/websocket-client.html
```

#### Debug Logging

```go
// Enable debug logging in Go
log.SetLevel(log.DebugLevel)

// Add custom logging
log.Printf("WebSocket message: %+v", message)
```

#### Browser Developer Tools

```javascript
// Monitor WebSocket in browser
// 1. Open DevTools → Network tab
// 2. Filter by "WS" (WebSocket)
// 3. Click on connection to see messages

// Log all WebSocket events
client.on("message_received", (message) => {
  console.log("Received:", message);
});

client.on("message_sent", (data) => {
  console.log("Sent:", data);
});
```

### 4. Production Deployment

#### CORS Configuration

```go
// Update CORS settings for production
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        allowedOrigins := []string{
            "https://your-domain.com",
            "https://app.your-domain.com",
        }
        for _, allowed := range allowedOrigins {
            if origin == allowed {
                return true
            }
        }
        return false
    },
}
```

#### Load Balancer Configuration

```nginx
# Nginx configuration for WebSocket
upstream websocket_backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
}

server {
    location /api/v1/ws/ {
        proxy_pass http://websocket_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### Redis Configuration

```bash
# Ensure Redis is running for scaling
redis-server --port 6379

# Test Redis connection
redis-cli ping
```

### 5. Performance Issues

#### High Memory Usage

```go
// Monitor connections and cleanup
func (h *Hub) periodicCleanup(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            h.cleanupInactiveClients()
            h.cleanupExpiredFocusSessions()
        case <-ctx.Done():
            return
        }
    }
}
```

**Solutions**:

1. **Monitor Connection Limits**: Set max connections per user
2. **Cleanup Inactive Sessions**: Regular cleanup of stale connections
3. **Optimize Message Size**: Avoid sending large payloads

#### Slow Message Delivery

```javascript
// Monitor message latency
const startTime = Date.now();
client.send("ping", { timestamp: startTime });

client.on("pong", (data) => {
  const latency = Date.now() - data.timestamp;
  console.log("Latency:", latency + "ms");
});
```

**Solutions**:

1. **Check Network**: Verify network connectivity
2. **Optimize Payloads**: Send minimal necessary data
3. **Use Message Batching**: Batch multiple updates when possible

### 6. Testing and Validation

#### Unit Tests

```go
// Test WebSocket message handling
func TestWebSocketMessage(t *testing.T) {
    hub := websocket.NewHub(nil)
    client := websocket.NewClient(hub, conn, userID, "test", "127.0.0.1")

    // Test message handling
    message := websocket.Message{
        Type: "ping",
        Payload: map[string]interface{}{"test": true},
    }

    client.handleMessage(message)
    // Assert expected behavior
}
```

#### Integration Tests

```javascript
// Test full WebSocket flow
describe("WebSocket Integration", () => {
  it("should connect and receive messages", async () => {
    const client = new WebSocketTodoClient({
      baseUrl: "ws://localhost:8080",
      token: validJWTToken,
    });

    await client.connect();

    const todo = await client.createTodo({
      task_name: "Test Task",
      project_id: projectId,
    });

    expect(todo).toBeDefined();
    expect(todo.task_name).toBe("Test Task");
  });
});
```

### 7. Common Error Codes

| Error Code                | Description                    | Solution               |
| ------------------------- | ------------------------------ | ---------------------- |
| `INVALID_MESSAGE`         | Malformed JSON message         | Check message format   |
| `MISSING_TODO_ID`         | Todo ID not provided           | Include valid UUID     |
| `INVALID_TODO_ID`         | Invalid UUID format            | Use proper UUID format |
| `FOCUS_SESSION_ERROR`     | Focus session operation failed | Check session state    |
| `AUTHENTICATION_REQUIRED` | Missing/invalid token          | Provide valid JWT      |

### 8. Health Check Commands

```bash
# Check server health
curl http://localhost:8080/health

# Check WebSocket status
curl http://localhost:8080/api/v1/ws/status \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Check connected users
curl http://localhost:8080/api/v1/ws/users \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Check focus sessions
curl http://localhost:8080/api/v1/ws/focus \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 9. Environment Variables

```bash
# WebSocket Configuration
export WS_MAX_CONNECTIONS=1000
export WS_MESSAGE_SIZE_LIMIT=1048576
export WS_PING_INTERVAL=54s
export WS_PONG_TIMEOUT=60s

# Redis Configuration (for scaling)
export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=
export REDIS_DB=0

# Focus Session Configuration
export FOCUS_TIMER_INTERVAL=60s
export FOCUS_SESSION_TIMEOUT=2h
```

### 10. Quick Fixes Checklist

When encountering issues, check:

- [ ] Server is running on correct port (8080)
- [ ] JWT token is valid and not expired
- [ ] WebSocket URL is correct
- [ ] CORS settings allow your origin
- [ ] Redis is running (for scaling features)
- [ ] Network firewall allows WebSocket connections
- [ ] Browser supports WebSocket protocol
- [ ] No proxy blocking WebSocket upgrade
- [ ] Server logs for detailed error messages

### 11. Getting Help

1. **Check Logs**: Enable debug logging for detailed information
2. **Test with Examples**: Use provided test scripts and HTML demo
3. **Verify API**: Test regular HTTP endpoints first
4. **Network Analysis**: Use browser DevTools or Wireshark
5. **Community**: Check documentation and examples

---

**Need more help?**

- Review the [WebSocket API Documentation](./WEBSOCKET_API_DOCUMENTATION.md)
- Try the [Interactive Demo](../examples/websocket-client.html)
- Run the [Test Script](../examples/websocket-test.js)
