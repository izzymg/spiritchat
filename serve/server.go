package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"spiritchat/data"
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
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(rw, "Internal server error")
		return
	}

	rw.WriteHeader(http.StatusOK)
	err = json.NewEncoder(rw).Encode(categories)
	if err != nil {
		log.Printf("failed to encode JSON response: %s", err)
	}
}

// NewServer stub todo
func NewServer(store *data.Store, address string) *Server {

	server := &Server{
		store: store,
		httpServer: http.Server{
			Addr: address,
		},
	}

	router := httprouter.New()
	router.GET("/:cat", server.GetCategories)

	server.httpServer.Handler = router
	return server
}
