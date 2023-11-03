package serve

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"spiritchat/data"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
)

const postFailMessage = "Sorry, an error occurred while saving your post"
const genericFailMessage = "Sorry, an error occurred while handling your request."

// Server stub todo
type Server struct {
	PostCooldownSeconds int
	cooldownMs          int
	store               data.Store
	httpServer          http.Server
}

func (server *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	server.httpServer.Handler.ServeHTTP(rw, req)
}

// Listen starts the server listening process until the context is cancelled (blocks).
func (server *Server) Listen(ctx context.Context) error {
	go server.httpServer.ListenAndServe()
	select {
	case <-ctx.Done():
		return server.httpServer.Shutdown(context.Background())
	}
}

// HandleGetCategories handles a GET request for information on categories.
func (server *Server) HandleGetCategories(ctx context.Context, req *request, respond respondFunc) {
	categories, err := server.store.GetCategories(ctx)
	if err != nil {
		respond(
			http.StatusInternalServerError, nil, genericFailMessage,
		)
		log.Println(err)
		return
	}

	respond(http.StatusOK, categories, "")
}

// HandleGetCatView handles a GET request for information on a single category.
func (server *Server) HandleGetCatView(ctx context.Context, req *request, respond respondFunc) {
	view, err := server.store.GetCatView(ctx, req.params.ByName("cat"))
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			respond(
				http.StatusNotFound,
				nil, err.Error(),
			)
			return
		}
		respond(
			http.StatusInternalServerError, nil, genericFailMessage,
		)
		log.Println(err)
		return
	}

	respond(http.StatusOK, view, "")
}

// HandleGetThreadView handles a GET request for information on a thread.
func (server *Server) HandleGetThreadView(ctx context.Context, req *request, respond respondFunc) {
	threadNum, err := strconv.Atoi(req.params.ByName("thread"))
	if err != nil {
		respond(http.StatusBadRequest, nil, "Invalid thread number")
		return
	}
	threadView, err := server.store.GetThreadView(ctx, req.params.ByName("cat"), threadNum)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			respond(http.StatusNotFound, nil, err.Error())
			return
		}
		respond(http.StatusInternalServerError, nil, genericFailMessage)
		log.Println(err)
		return
	}

	respond(http.StatusOK, threadView, "")
}

// HandleWritePost handles a POST request to post a new post.
func (server *Server) HandleWritePost(ctx context.Context, req *request, respond respondFunc) {
	catName := req.params.ByName("cat")
	threadNumber, err := strconv.Atoi(req.params.ByName("thread"))
	if err != nil {
		respond(
			http.StatusBadRequest,
			nil, "Invalid thread number",
		)
		return
	}

	// Decode body and write post
	userPost := &data.UserPost{}
	if req.rawRequest.Body == nil {
		respond(http.StatusBadRequest, nil, "no post provided")
		return
	}
	err = json.NewDecoder(req.rawRequest.Body).Decode(userPost)
	if err != nil {
		respond(http.StatusBadRequest, nil, "bad formatting")
		return
	}

	content, errMessage := data.CheckContent(userPost.Content)
	if len(errMessage) > 0 {
		respond(
			http.StatusBadRequest,
			nil, errMessage,
		)
		return
	}
	userPost.Content = content

	err = server.store.WritePost(ctx, catName, threadNumber, userPost)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			respond(http.StatusNotFound, nil, err.Error())
			return
		}
		respond(
			http.StatusInternalServerError, nil, postFailMessage,
		)
		log.Printf("Failed to save new post request: %s", err)
		return
	}

	respond(http.StatusOK, ok{Message: "Post submitted"}, "")
}

// Handle handleCORSPreflight pre-flighting
func handleCORSPreflight(allowedOrigin string) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		rw.Header().Set("Access-Control-Allow-Methods", "GET,POST")
		rw.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		rw.WriteHeader(http.StatusNoContent)
	}
}

func middlewareCORS(hand httprouter.Handle, allowedOrigin string) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		rw.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		hand(rw, req, params)
	}
}

func (s *Server) middlewareRateLimit(hand handlerFunc, ms int, resource string, limitedMessage string) handlerFunc {
	return func(ctx context.Context, req *request, respond respondFunc) {
		isLimited, err := s.store.IsRateLimited(req.ip, resource)
		if err != nil {
			respond(http.StatusInternalServerError, nil, "internal server error")
			log.Printf("Failed to fetch rate limit info: %s", err)
			return
		}

		if isLimited {
			respond(http.StatusTooManyRequests, nil, limitedMessage)
			return
		}

		err = s.store.RateLimit(req.ip, resource, ms)
		if err != nil {
			respond(http.StatusInternalServerError, nil, "internal server error")
			log.Printf("Failed to rate limit: %s", err)
			return
		}

		hand(ctx, req, respond)
	}
}

// ServerOptions configure the server.
type ServerOptions struct {
	Address             string
	CorsOriginAllow     string
	PostCooldownSeconds int
}

// NewServer stub todo
func NewServer(store data.Store, opts ServerOptions) *Server {

	server := &Server{
		store:      store,
		cooldownMs: opts.PostCooldownSeconds * 1000,
		httpServer: http.Server{
			Addr:              opts.Address,
			IdleTimeout:       time.Minute * 10,
			ReadHeaderTimeout: time.Second * 10,
		},
	}

	router := httprouter.New()
	router.GlobalOPTIONS = http.HandlerFunc(handleCORSPreflight(opts.CorsOriginAllow))
	router.GET("/v1", middlewareCORS(genHandler(server.HandleGetCategories), opts.CorsOriginAllow))
	router.GET("/v1/:cat", middlewareCORS(genHandler(server.HandleGetCatView), opts.CorsOriginAllow))
	router.POST("/v1/:cat/:thread", middlewareCORS(genHandler(server.HandleWritePost), opts.CorsOriginAllow))
	router.GET("/v1/:cat/:thread", middlewareCORS(genHandler(server.HandleGetThreadView), opts.CorsOriginAllow))

	server.httpServer.Handler = router
	return server
}
