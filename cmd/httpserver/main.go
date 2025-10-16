package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/shravanasati/shadowfax/middleware"
	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
	"github.com/shravanasati/shadowfax/router"
	"github.com/shravanasati/shadowfax/server"
)

const port = 42069

func headerAdder(next server.Handler) server.Handler {
	return func(r *request.Request) response.Response {
		resp := next(r)
		resp.WithHeader("X-Server", "shadowfax")
		return resp
	}
}

func userOnly(next server.Handler) server.Handler {
	return func(r *request.Request) response.Response {
		if r.Headers.Get("username") != "user" {
			return response.NewBaseResponse().WithStatusCode(response.StatusUnauthorized)
		}
		return next(r)
	}
}

func main() {
	app := router.NewRouter(&router.RouterOptions{
		EnableCors: true,
		CorsOptions: router.CorsOptions{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET"},
		},
	})
	app.Use(middleware.LoggingMiddlewareColored, headerAdder)

	subRouter := router.NewRouter(nil)
	subRouter.Use(userOnly)
	subRouter.Get("/*", func(r *request.Request) response.Response {
		return response.NewTextResponse("sub")
	})

	app.Handle("/sub", subRouter.Handler())
	app.Handle("/panic", func(r *request.Request) response.Response {
		panic("boom")
	})

	app.Get("/index", func(r *request.Request) response.Response {
		return response.NewHTMLResponse(`<h1>hullo</h1>`)
	})

	app.Get("/yourproblem", func(r *request.Request) response.Response {
		return response.
			NewTextResponse("your problem is not my problem\n").
			WithStatusCode(response.StatusBadRequest)
	})

	app.Handle("/myproblem", func(r *request.Request) response.Response {
		return response.
			NewTextResponse("woopsie, my bad\n").
			WithStatusCode(response.StatusInternalServerError)
	})

	app.Post("/httpbin/:x", func(r *request.Request) response.Response {
		xs := r.PathParams["x"]
		_, err := strconv.Atoi(xs)
		if err != nil {
			return response.NewBaseResponse().WithStatusCode(response.StatusBadRequest)
		}

		resp, err := http.Get("https://httpbin.org/stream/" + xs)
		if err != nil {
			fmt.Printf("Error fetching httpbin: %v\n", err)
			return response.
				NewBaseResponse().
				WithStatusCode(response.StatusInternalServerError)
		}

		fmt.Printf("httpbin response status: %s\n", resp.Status)
		fmt.Printf("httpbin response headers: %v\n", resp.Header)

		sr := response.NewStreamResponse(func(w io.Writer, setTrailer response.TrailerSetter) error {
			defer resp.Body.Close()

			buf := make([]byte, 1024)
			totalBytes := 0
			var allData bytes.Buffer
			for {
				n, err := resp.Body.Read(buf)
				if err == io.EOF {
					fmt.Printf("Finished reading httpbin response, total bytes: %d\n", totalBytes)
					// Set trailers before finishing
					hash := sha256.Sum256(allData.Bytes())
					setTrailer("X-Content-SHA256", hex.EncodeToString(hash[:]))
					setTrailer("X-Content-Length", strconv.Itoa(totalBytes))
					break
				}
				if err != nil {
					fmt.Printf("Error reading httpbin response: %v\n", err)
					return err
				}
				totalBytes += n
				allData.Write(buf[:n])
				fmt.Printf("Read %d bytes from httpbin, writing to client\n", n)
				w.Write(buf[:n])
			}

			return nil
		}, []string{"X-Content-Length", "X-Content-SHA256"})

		return sr
	})

	app.Get("/stream/:s", func(r *request.Request) response.Response {
		xs := r.PathParams["s"]
		s, err := strconv.Atoi(xs)
		if err != nil {
			return response.NewBaseResponse().WithStatusCode(response.StatusBadRequest)
		}

		sr := response.NewStreamResponse(func(w io.Writer, setTrailer response.TrailerSetter) error {
			ticker := time.NewTicker(time.Millisecond * 100)
			defer ticker.Stop()
			deadline := time.After(time.Duration(s) * time.Second)

			for {
				select {
				case <-deadline:
					return nil // finishes after 2s, pipe closes
				case t := <-ticker.C:
					fmt.Fprintf(w, "%v\n", t)
				}
			}
		}, nil)

		return sr
	})

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

	app.Post("/upload", func(r *request.Request) response.Response {
		body, err := r.Body()
		if err != nil {
			return response.NewTextResponse(err.Error()).WithStatusCode(response.StatusBadRequest)
		}
		content, err := io.ReadAll(body)
		fmt.Println(string(content), err)
		return response.NewBaseResponse()
	})

	app.Get("/file", func(r *request.Request) response.Response {
		f, err := os.Open(`./assets/vim.mp4`)
		if err != nil {
			return response.NewBaseResponse().WithStatusCode(response.StatusInternalServerError)
		}
		return response.NewFileResponse(f)
	})

	app.Delete("/api/:user", func(r *request.Request) response.Response {
		user := r.PathParams["user"]
		force := r.Query["force"]
		return response.
			NewTextResponse(fmt.Sprintf("user %s deleted with force=%s", user, force))
	})

	app.Get("/redirect", func(r *request.Request) response.Response {
		return response.NewRedirectResponse("https://google.com")
	})

	app.Handle("/api/*path", func(r *request.Request) response.Response {
		return response.NewTextResponse("all good, frfr\n").WithStatusCode(response.StatusOK)
	})

	server, err := server.Serve(server.ServerOpts{
		Address: ":42069",
		// Recovery: func(r any) response.Response {
		// 	return response.NewTextResponse(fmt.Sprintf("sowwy I fucked up due to %v :<)", r))
		// },
		ReadTimeout:      30 * time.Second,
		KeepAliveTimeout: 10 * time.Second,
		// WriteTimeout: time.Second,
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
