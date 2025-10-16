package middleware

import (
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
	"github.com/shravanasati/shadowfax/server"
)

// LoggingMiddleware provides basic logging without colors.
func LoggingMiddleware(next server.Handler) server.Handler {
	return server.Handler(func(r *request.Request) response.Response {
		now := time.Now()
		resp := next(r)
		log.Printf("%s %s %d in %s\n", r.Method, r.Target, resp.GetStatusCode(), time.Since(now))
		return resp
	})
}

// LoggingMiddlewareColored provides colored logging.
func LoggingMiddlewareColored(next server.Handler) server.Handler {
	methodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Background(lipgloss.Color("12")).Width(8).Align(lipgloss.Center)

	return server.Handler(func(r *request.Request) response.Response {
		now := time.Now()
		resp := next(r)

		// create styled method and status code
		statusCode := int(resp.GetStatusCode())
		statusStyle := getStatusCodeStyle(statusCode)
		styledStatus := statusStyle.Render(fmt.Sprintf("%d", statusCode))

		styledMethod := methodStyle.Render(r.Method)

		log.Printf("%s %s %s in %s\n", styledMethod, r.Target, styledStatus, time.Since(now))

		return resp
	})
}

// getStatusCodeStyle returns a lipgloss style for HTTP status codes
func getStatusCodeStyle(statusCode int) lipgloss.Style {
	switch {
	case statusCode >= 200 && statusCode < 300:
		// 2xx Success - Green
		return lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	case statusCode >= 300 && statusCode < 400:
		// 3xx Redirection - Yellow
		return lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	case statusCode >= 400 && statusCode < 500:
		// 4xx Client Error - Orange/Red
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	case statusCode >= 500:
		// 5xx Server Error - Bright Red
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	default:
		// Unknown status codes - White
		return lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	}
}
