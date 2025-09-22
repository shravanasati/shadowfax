# Shadowfax

<p align="center"> 
	<img src="gopher_shadowfax.png" height="300px">
</p>
<p align="center">
	<strong>YOU SHALL NOT PASS!</strong>
</p>

[![integration](https://github.com/shravanasati/shadowfax/actions/workflows/integration.yml/badge.svg)](https://github.com/shravanasati/shadowfax/actions/workflows/integration.yml)

A fast, lightweight HTTP/1.1 server built from scratch in Go. Shadowfax implements the HTTP protocol directly on top of TCP sockets, providing a complete web server solution with modern features and abstractions.

> **Note**: This project is built for educational purposes and learning HTTP internals. Not recommended for production usage.

## ✨ Features

### Core HTTP Implementation
- **From-scratch HTTP/1.1 parser** - Custom request parsing without `net/http`
- **Response writer** - Efficient response generation with proper HTTP formatting
- **Chunked transfer encoding** - Support for streaming responses with trailers
- **Content-Length handling** - Automatic body size detection and headers
- **Persistent Connections** - Supports persistent connections via `KeepAliveTimeout` configuration option

### Web Server Abstractions
- **Prefix-tree router** - Fast O(log n) routing with trie-based path matching
- **Dynamic path parameters** - Extract parameters like `/users/:id`
- **Wildcard routes** - Catch-all routes with `/*path` patterns  
- **Method-specific routing** - GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD
- **Automatic HEAD handling** - Auto-generates HEAD responses from GET handlers
- **Method not allowed detection** - Proper 405 responses for unsupported methods

### Advanced Features
- **Middleware support** - Composable request/response middleware chain
- **Panic recovery** - Graceful error handling with customizable recovery
- **Graceful shutdown** - Clean server termination with signal handling
- **Concurrent request handling** - Goroutine-per-request architecture
- **Query parameter parsing** - Easy access to URL query parameters
- **Multiple response types** - Text, JSON, HTML, File, and Stream responses

## 🚀 Quick Start

### Installation

```bash
go mod init your-project
go get github.com/shravanasati/shadowfax
```

### Basic Server

```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/shravanasati/shadowfax/internal/router"
    "github.com/shravanasati/shadowfax/internal/server"
    "github.com/shravanasati/shadowfax/internal/request"
    "github.com/shravanasati/shadowfax/internal/response"
)

func main() {
    // Create a new router
    app := router.NewRouter()
    
    // Add a simple route
    app.Get("/hello", func(r *request.Request) response.Response {
        return response.NewTextResponse("Hello, World!")
    })
    
    // Start the server
    srv, err := server.Serve(server.ServerOpts{
        Address:      ":8080",
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 10 * time.Second,
    }, app.Handler())
    
    if err != nil {
        log.Fatal(err)
    }
    defer srv.Close()
    
    log.Println("Server running on :8080")
    
    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    log.Println("Server stopped")
}
```

## 📖 Usage Guide

### Routing

#### Basic Routes

```go
app := router.NewRouter()

// HTTP methods
app.Get("/users", getUsersHandler)
app.Post("/users", createUserHandler) 
app.Put("/users/:id", updateUserHandler)
app.Delete("/users/:id", deleteUserHandler)
app.Patch("/users/:id", patchUserHandler)

// Handle any method
app.Handle("/webhook", webhookHandler)
```

#### Path Parameters

```go
app.Get("/users/:id", func(r *request.Request) response.Response {
    userID := r.PathParams["id"]
    return response.NewTextResponse("User ID: " + userID)
})

app.Get("/files/:category/:filename", func(r *request.Request) response.Response {
    category := r.PathParams["category"]
    filename := r.PathParams["filename"]
    
    return response.NewTextResponse(
        fmt.Sprintf("Category: %s, File: %s", category, filename)
    )
})
```

#### Wildcard Routes

```go
// Matches /api/v1/anything/here
app.Handle("/api/*path", func(r *request.Request) response.Response {
    fullPath := r.PathParams["path"]
    return response.NewTextResponse("API path: " + fullPath)
})
```

#### Subrouters

```go
// Create a subrouter with middleware
apiRouter := router.NewRouter()
apiRouter.Use(authMiddleware)

apiRouter.Get("/profile", profileHandler)
apiRouter.Post("/data", dataHandler)

// Mount the subrouter
app.Handle("/api", apiRouter.Handler())
```

### Request Handling

#### Query Parameters

```go
app.Get("/search", func(r *request.Request) response.Response {
    query := r.Query.Get("q")
    page := r.Query.Get("page")
    filters := r.Query["filter"] // Get all filter values
    
    return response.NewJSONResponse(map[string]any{
        "query": query,
        "page": page, 
        "filters": filters,
    })
})
```

#### Headers

```go
app.Post("/api/data", func(r *request.Request) response.Response {
    contentType := r.Headers.Get("content-type")
    userAgent := r.Headers.Get("user-agent")
    customHeader := r.Headers.Get("x-custom-header")
    
    if contentType != "application/json" {
        return response.NewBaseResponse().WithStatusCode(response.StatusBadRequest)
    }
    
    // Process request...
    return response.NewTextResponse("Data processed")
})
```

#### Request Body

```go
app.Post("/upload", func(r *request.Request) response.Response {
    body := r.Body()
    defer body.Close()
    
    data, err := io.ReadAll(body)
    if err != nil {
        return response.NewBaseResponse().WithStatusCode(response.StatusBadRequest)
    }
    
    // Process the body data...
    return response.NewTextResponse("Upload successful")
})
```

### Response Types

#### Text Response

```go
app.Get("/text", func(r *request.Request) response.Response {
    return response.NewTextResponse("Plain text response")
})
```

#### JSON Response

```go
app.Get("/api/user/:id", func(r *request.Request) response.Response {
    user := map[string]any{
        "id":    r.PathParams["id"],
        "name":  "John Doe", 
        "email": "john@example.com",
    }
    
    resp, err := response.NewJSONResponse(user)
    if err != nil {
        return response.
			NewBaseResponse().
			WithStatusCode(response.StatusInternalServerError)
    }
    
    return resp
})
```

#### HTML Response

```go
app.Get("/", func(r *request.Request) response.Response {
    html := `
    <!DOCTYPE html>
    <html>
    <head><title>Shadowfax</title></head>
    <body><h1>Welcome to Shadowfax!</h1></body>
    </html>
    `
    return response.NewHTMLResponse(html)
})
```

#### File Response

```go
app.Get("/download/:filename", func(r *request.Request) response.Response {
    filename := r.PathParams["filename"]
    
    file, err := os.Open("./uploads/" + filename)
    if err != nil {
        return response.NewBaseResponse().WithStatusCode(response.StatusNotFound)
    }
    
    return response.NewFileResponse(file).
        WithHeader("Content-Type", "application/octet-stream").
        WithHeader("Content-Disposition", "attachment; filename="+filename)
})
```

#### Streaming Response

```go
app.Get("/stream/:duration", func(r *request.Request) response.Response {
    duration, _ := strconv.Atoi(r.PathParams["duration"])
    
    streamFunc := func(w io.Writer, setTrailer response.TrailerSetter) error {
        ticker := time.NewTicker(100 * time.Millisecond)
        defer ticker.Stop()
        
        deadline := time.After(time.Duration(duration) * time.Second)
        
        for {
            select {
            case <-deadline:
                setTrailer("X-Stream-End", time.Now().String())
                return nil
            case t := <-ticker.C:
                fmt.Fprintf(w, "Time: %v\n", t)
            }
        }
    }
    
    return response.
		NewStreamResponse(streamFunc, []string{"X-Stream-End"})
})
```

#### Custom Status Codes and Headers

```go
app.Get("/custom", func(r *request.Request) response.Response {
    return response.NewTextResponse("Custom response").
        WithStatusCode(response.StatusCreated).
        WithHeader("X-Custom", "value").
        WithHeaders(map[string]string{
            "X-API-Version": "1.0",
            "X-Rate-Limit":  "100",
        })
})
```

### Middleware

#### Basic Middleware

```go
func loggingMiddleware(next server.Handler) server.Handler {
    return func(r *request.Request) response.Response {
        start := time.Now()
        resp := next(r)
        
        fmt.Printf("%s %s %d - %v\n", 
            r.Method, r.Target, resp.GetStatusCode(), time.Since(start))
        
        return resp
    }
}

app.Use(loggingMiddleware)
```

#### Authentication Middleware

```go
func authMiddleware(next server.Handler) server.Handler {
    return func(r *request.Request) response.Response {
        token := r.Headers.Get("Authorization")
        
        if !isValidToken(token) {
            return response.NewBaseResponse().
                WithStatusCode(response.StatusUnauthorized)
        }
        
        return next(r)
    }
}

// Apply to specific routes
protectedRouter := router.NewRouter()
protectedRouter.Use(authMiddleware)
protectedRouter.Get("/profile", profileHandler)

app.Handle("/api", protectedRouter.Handler())
```

#### CORS Middleware

```go
func corsMiddleware(next server.Handler) server.Handler {
    return func(r *request.Request) response.Response {
        resp := next(r)
        
        return resp.
            WithHeader("Access-Control-Allow-Origin", "*").
            WithHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE").
            WithHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
    }
}

app.Use(corsMiddleware)
```

### Error Handling

#### Server Configuration

Configure server timeouts and recovery options:

```go
srv, err := server.Serve(server.ServerOpts{
    Address:      ":8080",
    ReadTimeout:  30 * time.Second,  // Maximum time to read the entire request
    WriteTimeout: 10 * time.Second,  // Maximum time to write the response
    Recovery: func(r any) response.Response {
        log.Printf("Panic recovered: %v", r)
        return response.NewTextResponse("Internal Server Error").
            WithStatusCode(response.StatusInternalServerError)
    },
}, app.Handler())
```

**Configuration Options:**
- `Address` - Server bind address (e.g., ":8080", "localhost:3000")
- `ReadTimeout` - Maximum duration for reading the entire request (including body)
- `WriteTimeout` - Maximum duration for writing the response
- `KeepAliveTimeout` - Maximum duration for idle connection. Defaults to 0, which disables keep-alive.
- `Recovery` - Custom panic recovery function

#### Custom 404 Handler

```go
app.NotFound(func(r *request.Request) response.Response {
    return response.NewHTMLResponse(`
        <h1>Page Not Found</h1>
        <p>The page you're looking for doesn't exist.</p>
    `).WithStatusCode(response.StatusNotFound)
})
```

#### Panic Recovery

```go
srv, err := server.Serve(server.ServerOpts{
    Address:      ":8080",
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 10 * time.Second,
    Recovery: func(r any) response.Response {
        log.Printf("Panic recovered: %v", r)
        return response.NewTextResponse("Internal Server Error").
            WithStatusCode(response.StatusInternalServerError)
    },
}, app.Handler())
```

### Complete Example

```go
package main

import (
    "fmt"
    "log"
    "os"
    "os/signal"
    "strconv"
    "syscall"
    "time"
    
    "github.com/shravanasati/shadowfax/internal/request"
    "github.com/shravanasati/shadowfax/internal/response" 
    "github.com/shravanasati/shadowfax/internal/router"
    "github.com/shravanasati/shadowfax/internal/server"
)

func main() {
    app := router.NewRouter()
    
    // Global middleware
    app.Use(loggingMiddleware, corsMiddleware)
    
    // Static routes
    app.Get("/", homeHandler)
    app.Get("/health", healthHandler)
    
    // API routes with parameters
    app.Get("/api/users/:id", getUserHandler)
    app.Post("/api/users", createUserHandler)
    app.Delete("/api/users/:id", deleteUserHandler)
    
    // File operations
    app.Get("/files/*path", fileHandler)
    app.Post("/upload", uploadHandler)
    
    // Streaming endpoint
    app.Get("/stream/:seconds", streamHandler)
    
    // Protected routes
    adminRouter := router.NewRouter()
    adminRouter.Use(authMiddleware)
    adminRouter.Get("/stats", adminStatsHandler)
    adminRouter.Post("/config", adminConfigHandler)
    app.Handle("/admin", adminRouter.Handler())
    
    // Custom error handlers
    app.NotFound(notFoundHandler)
    
    // Start server
    srv, err := server.Serve(server.ServerOpts{
        Address:      ":8080",
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 10 * time.Second,
        Recovery:     panicHandler,
    }, app.Handler())
    
    if err != nil {
        log.Fatal(err)
    }
    defer srv.Close()
    
    log.Println("🚀 Shadowfax server running on :8080")
    
    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    log.Println("🛑 Server stopped gracefully")
}

func homeHandler(r *request.Request) response.Response {
    return response.NewHTMLResponse(`
        <h1>Welcome to Shadowfax</h1>
        <p>A fast HTTP/1.1 server built from scratch!</p>
    `)
}

func healthHandler(r *request.Request) response.Response {
    return response.NewJSONResponse(map[string]any{
        "status": "healthy",
        "time":   time.Now().Unix(),
    })
}

func getUserHandler(r *request.Request) response.Response {
    userID := r.PathParams["id"]
    
    user := map[string]any{
        "id":    userID,
        "name":  "User " + userID,
        "email": fmt.Sprintf("user%s@example.com", userID),
    }
    
    resp, _ := response.NewJSONResponse(user)
    return resp
}

func loggingMiddleware(next server.Handler) server.Handler {
    return func(r *request.Request) response.Response {
        start := time.Now()
        resp := next(r)
        
        fmt.Printf("%s %s %d - %v\n", 
            r.Method, r.Target, resp.GetStatusCode(), time.Since(start))
        
        return resp
    }
}

func corsMiddleware(next server.Handler) server.Handler {
    return func(r *request.Request) response.Response {
        resp := next(r)
        return resp.WithHeader("Access-Control-Allow-Origin", "*")
    }
}
```

## 🏗️ Architecture

### Request Flow

1. **TCP Connection** - Accept incoming connections
2. **HTTP Parsing** - Parse HTTP/1.1 request line and headers  
3. **Routing** - Match path against trie-based router
4. **Middleware Chain** - Execute middleware in order
5. **Handler Execution** - Call matched route handler
6. **Response Generation** - Create and write HTTP response
7. **Connection Management** - Handle keep-alive or close

### Router Implementation

Shadowfax uses a **prefix tree (trie)** for efficient route matching:

- **Static segments** - Exact string matches (`/users`)
- **Parameter segments** - Dynamic captures (`/users/:id`) 
- **Wildcard segments** - Catch-all matches (`/files/*path`)

Route precedence: Static → Parameters → Wildcards

### Concurrency Model

- **Goroutine per request** - Each connection handled concurrently
- **Shared router** - Thread-safe routing with immutable trie
- **Graceful shutdown** - Clean termination of active connections

## 🧪 Testing

Run the test suite:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

## 📜 License

This project is licensed under the MIT License - see the [LICENSE.txt](LICENSE.txt) file for details.
