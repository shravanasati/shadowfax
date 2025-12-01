package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const port = 42069

func main() {
	mux := http.NewServeMux()

	// JSON endpoint
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		data := map[string]any{
			"hello": 1,
			"hi":    "bye",
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// Template endpoint with path parameter
	mux.HandleFunc("/template/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract name from path
		path := strings.TrimPrefix(r.URL.Path, "/template/")
		name := path
		if name == "" {
			name = "World"
		}

		tmplStr := `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { color: #2c3e50; }
        .info { background: #ecf0f1; padding: 20px; border-radius: 5px; }
        .params { margin-top: 20px; }
        .param { background: #3498db; color: white; padding: 5px 10px; margin: 5px; border-radius: 3px; display: inline-block; }
    </style>
</head>
<body>
    <h1 class="header">Hello, {{.Name}}!</h1>
    <div class="info">
        <p><strong>Server:</strong> Shadowfax HTTP Server</p>
        <p><strong>Template Engine:</strong> Go html/template</p>
        <p><strong>Timestamp:</strong> {{.Timestamp}}</p>
        <p><strong>Request Method:</strong> {{.Method}}</p>
        <p><strong>Request Path:</strong> {{.Path}}</p>
    </div>
    {{if .QueryParams}}
    <div class="params">
        <h3>Query Parameters:</h3>
        {{range $key, $value := .QueryParams}}
        <span class="param">{{$key}}: {{$value}}</span>
        {{end}}
    </div>
    {{end}}
    <p><em>This page was generated using Shadowfax's template response system.</em></p>
</body>
</html>`

		// Collect query parameters
		queryParams := make(map[string]string)
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				queryParams[key] = values[0]
			}
		}

		data := map[string]interface{}{
			"Title":       fmt.Sprintf("Welcome %s", name),
			"Name":        name,
			"Timestamp":   time.Now().Format("2006-01-02 15:04:05 MST"),
			"Method":      r.Method,
			"Path":        r.URL.Path,
			"QueryParams": queryParams,
		}

		tmpl, err := template.New("page").Parse(tmplStr)
		if err != nil {
			http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		log.Println("Server started on port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("Server gracefully stopped")
}
