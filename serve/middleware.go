package serve

import (
	"context"
	"log"
	"net/http"
)

func (s *Server) middlewareCORS(hand handlerFunc, allowedOrigin string) handlerFunc {
	return func(ctx context.Context, req *request, res *response) {
		res.rw.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		hand(ctx, req, res)
	}
}

func (s *Server) middlewareRateLimit(hand handlerFunc, ms int, resource string) handlerFunc {
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

		hand(ctx, req, res)
	}
}
