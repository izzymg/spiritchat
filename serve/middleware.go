package serve

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

func (s *Server) middlewareCORS(next handlerFunc, allowedOrigin string) handlerFunc {
	return func(ctx context.Context, req *request, res *response) {
		res.rw.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		next(ctx, req, res)
	}
}

func (s *Server) middlewareRateLimit(next handlerFunc, ms int, resource string) handlerFunc {
	return func(ctx context.Context, req *request, res *response) {
		isLimited, err := s.store.IsRateLimited(req.ip, resource)
		if err != nil {
			res.Respond(http.StatusInternalServerError, nil, "internal server error")
			log.Printf("Failed to fetch rate limit info: %s", err)
			return
		}

		if isLimited {
			res.Respond(http.StatusTooManyRequests, nil, "Rate limited")
			return
		}

		err = s.store.RateLimit(req.ip, resource, ms)
		if err != nil {
			res.Respond(http.StatusInternalServerError, nil, "internal server error")
			log.Printf("Failed to rate limit: %s", err)
			return
		}

		next(ctx, req, res)
	}
}

func (s *Server) middlewareRequireLogin(next handlerFunc) handlerFunc {
	return func(ctx context.Context, req *request, res *response) {
		token := req.header.Get("Authorization")
		if len(token) < 1 {
			res.Respond(http.StatusForbidden, nil, "no access token")
			return
		}
		user, err := s.auth.GetUserFromToken(ctx, token)
		if err != nil {
			res.Respond(http.StatusForbidden, nil, fmt.Sprintf("look up user failure: %s", err))
			return
		}
		if user == nil {
			res.Respond(http.StatusNotFound, nil, "no user")
			return
		}
		req.user = &requestUser{
			Username: user.Username,
			Email:    user.Email,
		}
		next(ctx, req, res)
	}
}
