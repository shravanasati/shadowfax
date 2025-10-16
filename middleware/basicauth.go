package middleware

import (
	"encoding/base64"
	"strings"

	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
	"github.com/shravanasati/shadowfax/router"
	"github.com/shravanasati/shadowfax/server"
)

type Account struct {
	Username string
	Password string
}

func BasicAuthMiddleware(accounts []Account) router.Middleware {
	accountMap := make(map[string]string)
	for _, acc := range accounts {
		accountMap[acc.Username] = acc.Password
	}

	return func(next server.Handler) server.Handler {
		return func(r *request.Request) response.Response {
			auth := r.Headers.Get("Authorization")

			if !strings.HasPrefix(auth, "Basic ") {
				return response.NewBaseResponse().
					WithStatusCode(response.StatusUnauthorized).
					WithHeader("www-authenticate", `Basic realm="Restricted"`)
			}

			payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
			if err != nil {
				return response.NewTextResponse("Invalid authorization header").
					WithStatusCode(response.StatusBadRequest)

			}

			parts := strings.SplitN(string(payload), ":", 2)
			if len(parts) != 2 {
				return response.NewTextResponse("Invalid authorization header").
					WithStatusCode(response.StatusBadRequest)
			}

			user, pass := parts[0], parts[1]
			actualPass, ok := accountMap[user]
			if !ok || actualPass != pass {
				return response.NewBaseResponse().
					WithHeader("WWW-Authenticate", `Basic realm="Restricted"`).
					WithStatusCode(response.StatusUnauthorized)
			}

			return next(r)
		}
	}
}
