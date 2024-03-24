package serve

import (
	"context"
	"fmt"
	"net/http"
)

func (s *Server) middlewareCORS(next handlerFunc, allowedOrigin string) handlerFunc {
	return func(ctx context.Context, req *request, res *response) {
		res.rw.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		res.rw.Header().Set("Access-Control-Allow-Headers", "Authorization")
		next(ctx, req, res)
	}
}

func (s *Server) middlewareRequireLogin(next handlerFunc) handlerFunc {
	return func(ctx context.Context, req *request, res *response) {
		token := req.header.Get("Authorization")
		if len(token) < 1 {
			res.Respond(http.StatusUnauthorized, nil, "no access token")
			return
		}
		user, err := s.auth.GetUserFromToken(ctx, token)
		if err != nil {
			res.Respond(http.StatusUnauthorized, nil, fmt.Sprintf("look up user failure: %s", err))
			return
		}
		if user == nil {
			res.Respond(http.StatusNotFound, nil, "no user")
			return
		}
		if !user.IsVerified {
			res.Respond(http.StatusUnauthorized, nil, "please verify your account")
			return
		}
		req.user = user
		next(ctx, req, res)
	}
}
