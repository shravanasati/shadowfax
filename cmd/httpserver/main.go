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

	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
	"github.com/shravanasati/shadowfax/internal/router"
	"github.com/shravanasati/shadowfax/internal/server"
)

const port = 42069

func loggingMiddleware(next server.Handler) server.Handler {
	return server.Handler(func(r *request.Request) response.Response {
		now := time.Now()
		resp := next(r)
		fmt.Printf("%s %s %d in %s\n", r.Method, r.Target, resp.GetStatusCode(), time.Since(now))
		return resp
	})
}

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
	app := router.NewRouter()
	app.Use(loggingMiddleware, headerAdder)

	fuckRouter := router.NewRouter()
	fuckRouter.Use(userOnly)
	fuckRouter.Get("/*", func(r *request.Request) response.Response {
		return response.NewTextResponse("fuck")
	})

	app.Handle("/fuck", fuckRouter.Handler())
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

	app.Get("/file", func(r *request.Request) response.Response {
		f, err := os.Open(`./assets/vim.mp4`)
		if err != nil {
			return response.NewBaseResponse().WithStatusCode(response.StatusInternalServerError)
		}
		return response.NewFileResponse(f).WithHeader("content-type", "video/mp4")
	})

	app.Delete("/api/:user", func(r *request.Request) response.Response {
		user := r.PathParams["user"]
		force := r.Query["force"]
		return response.
			NewTextResponse(fmt.Sprintf("user %s deleted with force=%s", user, force))
	})

	app.Handle("/api/*path", func(r *request.Request) response.Response {
		return response.NewTextResponse("all good, frfr\n").WithStatusCode(response.StatusOK)
	})

	server, err := server.Serve(port, app.Handler())

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
