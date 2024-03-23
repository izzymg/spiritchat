package serve

import (
	"context"
	"errors"
	"log"
	"net/http"
	"spiritchat/auth"
	"spiritchat/data"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
)

const postFailMessage = "Sorry, an error occurred while saving your post"
const genericFailMessage = "Sorry, an error occurred while handling your request."

var errBadThreadNumber = errors.New("invalid thread number")

// Server stub todo
type Server struct {
	PostCooldownSeconds int
	postCooldownMs      int
	store               data.Store
	auth                auth.Auth
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

// handleGetCategories handles a GET request for information on categories.
func (server *Server) handleGetCategories(ctx context.Context, req *request, res *response) {
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

// handleGetCategoryView handles a GET request for information on a single category.
func (server *Server) handleGetCategoryView(ctx context.Context, req *request, res *response) {
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

// handleGetThreadView handles a GET request for information on a thread.
func (server *Server) handleGetThreadView(ctx context.Context, req *request, res *response) {
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

// HandleSignUp handles a POST request for a sign up.
func (server *Server) handleSignUp(ctx context.Context, req *request, res *response) {
	incSignUp, err := getIncomingSignup(req.rawRequest.Body)
	if err != nil {
		res.Respond(http.StatusBadRequest, nil, err.Error())
		return
	}
	err = incSignUp.Sanitize()
	if err != nil {
		res.Respond(http.StatusBadRequest, nil, err.Error())
		return
	}

	data, err := server.auth.RequestSignUp(ctx, incSignUp.Username, incSignUp.Email, incSignUp.Password)
	if err != nil {
		res.Respond(http.StatusBadRequest, nil, err.Error())
		return
	}
	res.Respond(http.StatusOK, data, "success")
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
func getIncomingReplyParams(req *request) (*createPostParams, error) {
	threadNumber, err := strconv.Atoi(req.params.ByName("thread"))
	if err != nil {
		return nil, errBadThreadNumber
	}

	return &createPostParams{
		categoryTag:  req.params.ByName("cat"),
		threadNumber: threadNumber,
	}, nil
}

// handleCreatePost handles a POST request to post a new post.
func (server *Server) handleCreatePost(ctx context.Context, req *request, res *response) {

	params, err := getIncomingReplyParams(req)
	if err != nil {
		res.Respond(http.StatusBadRequest, nil, err.Error())
		return
	}

	incomingReply, err := getIncomingReply(req.rawRequest.Body)
	if err != nil {
		res.Respond(http.StatusBadRequest, nil, err.Error())
		return
	}

	err = incomingReply.Sanitize(params.isThread())
	if err != nil {
		res.Respond(http.StatusBadRequest, nil, err.Error())
		return
	}

	err = server.store.WritePost(
		ctx,
		params.categoryTag,
		params.threadNumber,
		incomingReply.Subject,
		incomingReply.Content,
		req.user.Username,
		req.user.Email,
		req.ip,
	)
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

func (server *Server) handleGetConfig(ctx context.Context, req *request, res *response) {
	res.Respond(http.StatusOK, ConfigResponse{
		Cooldown: server.postCooldownMs,
	}, "")
}

// Handle handleCORSPreflight pre-flighting
func handleCORSPreflight(allowedOrigin string) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		rw.Header().Set("Access-Control-Allow-Methods", "GET,POST")
		rw.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		rw.WriteHeader(http.StatusNoContent)
	}
}

// ServerOptions configure the server.
type ServerOptions struct {
	Address             string
	CorsOriginAllow     string
	PostCooldownSeconds int
}

// NewServer stub todo
func NewServer(store data.Store, auth auth.Auth, opts ServerOptions) *Server {

	server := &Server{
		store:          store,
		postCooldownMs: opts.PostCooldownSeconds * 1000,
		httpServer: http.Server{
			Addr:              opts.Address,
			IdleTimeout:       time.Minute * 10,
			ReadHeaderTimeout: time.Second * 10,
		},
		auth: auth,
	}

	router := httprouter.New()
	router.GlobalOPTIONS = http.HandlerFunc(
		handleCORSPreflight(opts.CorsOriginAllow),
	)

	router.GET(
		"/v1/categories",
		makeHandler(
			server.middlewareCORS(
				server.middlewareRateLimit(server.handleGetCategories, 100, "get-cats"),
				opts.CorsOriginAllow,
			),
		),
	)
	router.GET(
		"/v1/categories/:cat",
		makeHandler(
			server.middlewareCORS(
				server.middlewareRateLimit(server.handleGetCategoryView, 100, "get-catview"), opts.CorsOriginAllow,
			),
		),
	)
	router.POST(
		"/v1/categories/:cat/:thread",
		makeHandler(
			server.middlewareCORS(
				server.middlewareRequireLogin(
					server.middlewareRateLimit(
						server.handleCreatePost, server.postCooldownMs, "post-post"),
				),
				opts.CorsOriginAllow,
			),
		),
	)
	router.GET(
		"/v1/categories/:cat/:thread",
		makeHandler(
			server.middlewareCORS(
				server.middlewareRateLimit(server.handleGetThreadView, 100, "get-threadview"),
				opts.CorsOriginAllow,
			),
		),
	)
	router.POST(
		"/v1/signup",
		makeHandler(
			server.middlewareCORS(
				server.middlewareRateLimit(server.handleSignUp, 100, "post-signup"),
				opts.CorsOriginAllow,
			),
		),
	)
	router.GET(
		"/v1/config",
		makeHandler(
			server.middlewareCORS(
				server.handleGetConfig,
				opts.CorsOriginAllow,
			),
		),
	)

	server.httpServer.Handler = router
	return server
}
