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

// Server stub todo
type Server struct {
	store      *data.Store
	httpServer http.Server
}

// Listen starts the server listening process until the context is cancelled (blocks).
func (server *Server) Listen(ctx context.Context) error {
	go server.httpServer.ListenAndServe()
	select {
	case <-ctx.Done():
		return server.httpServer.Shutdown(context.Background())
	}
}

// GetCategories handles a GET request for information on categories.
func (server *Server) GetCategories(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	ctx, cancel := context.WithTimeout(req.Context(), time.Second*10)
	defer cancel()
	categories, err := server.store.GetCategories(ctx)
	if err != nil {
		internalError("Sorry, an error occurred while fetching categories")(rw, req)
		log.Println(err)
		return
	}

	err = json.NewEncoder(rw).Encode(categories)
	if err != nil {
		log.Printf("failed to encode JSON response: %s", err)
	}
}

// GetCatView handles a GET request for information on a single category.
func (server *Server) GetCatView(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	ctx, cancel := context.WithTimeout(req.Context(), time.Second*10)
	defer cancel()
	view, err := server.store.GetCatView(ctx, params.ByName("cat"))
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			notFound(err.Error())(rw, req)
			return
		}
		internalError("Sorry, an error occurred while fetching the category's threads")(rw, req)
		log.Println(err)
		return
	}

	err = json.NewEncoder(rw).Encode(view)
	if err != nil {
		log.Printf("failed to encode JSON response: %s", err)
	}
}

// GetThread handles a GET request for information on a thread.
func (server *Server) GetThread(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	ctx, cancel := context.WithTimeout(req.Context(), time.Second*10)
	defer cancel()

	threadNum, err := strconv.Atoi(params.ByName("thread"))
	if err != nil {
		notFound("Invalid thread number")(rw, req)
		return
	}
	thread, err := server.store.GetThread(ctx, params.ByName("cat"), threadNum)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			notFound(err.Error())(rw, req)
			return
		}
		internalError("Sorry, an error occurred while fetching the thread")(rw, req)
		log.Println(err)
		return
	}

	err = json.NewEncoder(rw).Encode(thread)
	if err != nil {
		log.Printf("failed to encode JSON response: %s", err)
	}
}

// NewServer stub todo
func NewServer(store *data.Store, address string) *Server {

	server := &Server{
		store: store,
		httpServer: http.Server{
			Addr:              address,
			IdleTimeout:       time.Minute * 10,
			ReadHeaderTimeout: time.Second * 10,
		},
	}

	router := httprouter.New()
	router.GET("/v1", server.GetCategories)
	router.GET("/v1/:cat", server.GetCatView)
	router.GET("/v1/:cat/:thread", server.GetThread)

	server.httpServer.Handler = router
	return server
}
