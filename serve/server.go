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

var errBadThreadNumber = errors.New("invalid thread number")
var errNoPost = errors.New("no post provided")
var errBadJson = errors.New("bad JSON")

// Server stub todo
type Server struct {
	PostCooldownSeconds int
	postCooldownMs      int
	store               data.Store
	httpServer          http.Server
}

func (server *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	server.httpServer.Handler.ServeHTTP(rw, req)
}

// Listen starts the server listening process until the context is cancelled (blocks).
func (server *Server) Listen(ctx context.Context) error {
	go server.httpServer.ListenAndServe()
	<-ctx.Done()
	return server.httpServer.Shutdown(context.Background())
}

// HandleGetCategories handles a GET request for information on categories.
func (server *Server) HandleGetCategories(ctx context.Context, req *request, res *response) {
	categories, err := server.store.GetCategories(ctx)
	if err != nil {
		res.Respond(
			http.StatusInternalServerError, nil, genericFailMessage,
		)
		log.Println(err)
		return
	}

	res.Respond(http.StatusOK, categories, "")
}

// HandleGetCategoryView handles a GET request for information on a single category.
func (server *Server) HandleGetCategoryView(ctx context.Context, req *request, res *response) {
	view, err := server.store.GetCategoryView(ctx, req.params.ByName("cat"))
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			res.Respond(
				http.StatusNotFound,
				nil, err.Error(),
			)
			return
		}
		res.Respond(
			http.StatusInternalServerError, nil, genericFailMessage,
		)
		log.Println(err)
		return
	}

	res.Respond(http.StatusOK, view, "")
}

// HandleGetThreadView handles a GET request for information on a thread.
func (server *Server) HandleGetThreadView(ctx context.Context, req *request, res *response) {
	threadNum, err := strconv.Atoi(req.params.ByName("thread"))
	if err != nil {
		res.Respond(http.StatusBadRequest, nil, "Invalid thread number")
		return
	}
	threadView, err := server.store.GetThreadView(ctx, req.params.ByName("cat"), threadNum)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			res.Respond(http.StatusNotFound, nil, err.Error())
			return
		}
		res.Respond(http.StatusInternalServerError, nil, genericFailMessage)
		log.Println(err)
		return
	}

	res.Respond(http.StatusOK, threadView, "")
}

// Data about a post creation request
type createPostParams struct {
	categoryTag  string
	threadNumber int
}

func (cpp createPostParams) isThread() bool {
	return cpp.threadNumber == 0
}

// Gets parameters for a post creation request
func getCreatePostParams(req *request) (*createPostParams, error) {
	threadNumber, err := strconv.Atoi(req.params.ByName("thread"))
	if err != nil {
		return nil, errBadThreadNumber
	}

	return &createPostParams{
		categoryTag:  req.params.ByName("cat"),
		threadNumber: threadNumber,
	}, nil
}

// Decodes a create post request into an UNSAFE user post
func decodeCreatePost(req *request) (*data.UnsafeUserPost, error) {
	unsafePost := &data.UnsafeUserPost{}
	if req.rawRequest.Body == nil {
		return nil, errNoPost
	}
	err := json.NewDecoder(req.rawRequest.Body).Decode(unsafePost)
	if err != nil {
		return nil, errBadJson
	}
	return unsafePost, nil
}

// HandleCreatePost handles a POST request to post a new post.
func (server *Server) HandleCreatePost(ctx context.Context, req *request, res *response) {
	params, err := getCreatePostParams(req)
	if err != nil {
		res.Respond(http.StatusBadRequest, nil, err.Error())
		return
	}

	unsafePost, err := decodeCreatePost(req)
	if err != nil {
		res.Respond(http.StatusBadRequest, nil, err.Error())
		return
	}

	safePost, err := data.SanitizeUnsafe(unsafePost, params.isThread())
	if err != nil {
		res.Respond(http.StatusBadRequest, nil, err.Error())
		return
	}

	err = server.store.WritePost(ctx, params.categoryTag, params.threadNumber, safePost)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			res.Respond(http.StatusNotFound, nil, err.Error())
			return
		}
		res.Respond(
			http.StatusInternalServerError, nil, postFailMessage,
		)
		log.Printf("Failed to save new post request: %s", err)
		return
	}

	res.Respond(http.StatusOK, ok{Message: "post submitted"}, "")
}

type ConfigResponse struct {
	Cooldown int `json:"cooldown"`
}

func (server *Server) HandleGetConfig(ctx context.Context, req *request, res *response) {
	res.Respond(http.StatusOK, ConfigResponse{
		Cooldown: server.postCooldownMs,
	}, "")
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

// ServerOptions configure the server.
type ServerOptions struct {
	Address             string
	CorsOriginAllow     string
	PostCooldownSeconds int
}

// NewServer stub todo
func NewServer(store data.Store, opts ServerOptions) *Server {

	server := &Server{
		store:          store,
		postCooldownMs: opts.PostCooldownSeconds * 1000,
		httpServer: http.Server{
			Addr:              opts.Address,
			IdleTimeout:       time.Minute * 10,
			ReadHeaderTimeout: time.Second * 10,
		},
	}

	router := httprouter.New()
	router.GlobalOPTIONS = http.HandlerFunc(
		handleCORSPreflight(opts.CorsOriginAllow),
	)

	router.GET(
		"/v1/categories",
		makeHandler(
			server.middlewareCORS(
				server.middlewareRateLimit(server.HandleGetCategories, 100, "get-cats"),
				opts.CorsOriginAllow,
			),
		),
	)
	router.GET(
		"/v1/categories/:cat",
		makeHandler(
			server.middlewareCORS(
				server.middlewareRateLimit(server.HandleGetCategoryView, 100, "get-catview"), opts.CorsOriginAllow,
			),
		),
	)
	router.POST(
		"/v1/categories/:cat/:thread",
		makeHandler(
			server.middlewareCORS(
				server.middlewareRateLimit(server.HandleCreatePost, server.postCooldownMs, "post-post"),
				opts.CorsOriginAllow,
			),
		),
	)
	router.GET(
		"/v1/categories/:cat/:thread",
		makeHandler(
			server.middlewareCORS(
				server.middlewareRateLimit(server.HandleGetThreadView, 100, "get-threadview"),
				opts.CorsOriginAllow,
			),
		),
	)
	router.GET(
		"/v1/config",
		makeHandler(
			server.middlewareCORS(
				server.HandleGetConfig,
				opts.CorsOriginAllow,
			),
		),
	)

	server.httpServer.Handler = router
	return server
}
