package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
	"github.com/shravanasati/shadowfax/router"
	"github.com/shravanasati/shadowfax/server"
)

const port = 42069

func main() {
	app := router.NewRouter(nil)

	app.Get("/json", func(r *request.Request) response.Response {
		jr, err := response.NewJSONResponse(map[string]any{
			"hello": 1,
			"hi":    "bye",
		})

		if err != nil {
			return response.NewBaseResponse().WithStatusCode(response.StatusInternalServerError)
		}

		return jr
	})

	app.Get("/template/:name", func(r *request.Request) response.Response {
		name := r.PathParams["name"]
		if name == "" {
			name = "World"
		}

		template := `<!DOCTYPE html>
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
		for key, values := range r.Query {
			if len(values) > 0 {
				queryParams[key] = values[0]
			}
		}

		data := map[string]interface{}{
			"Title":       fmt.Sprintf("Welcome %s", name),
			"Name":        name,
			"Timestamp":   time.Now().Format("2006-01-02 15:04:05 MST"),
			"Method":      r.Method,
			"Path":        r.Target,
			"QueryParams": queryParams,
		}

		tr, err := response.NewTemplateResponse(template, data)
		if err != nil {
			return response.NewTextResponse("Template error: " + err.Error()).WithStatusCode(response.StatusInternalServerError)
		}

		return tr
	})

	server, err := server.Serve(server.ServerOpts{
		Address: ":42069",
	}, app.Handler())

	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
