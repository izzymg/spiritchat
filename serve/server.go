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

// HandleGetCategories handles a GET request for information on categories.
func (server *Server) HandleGetCategories(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
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

// HandleGetCatView handles a GET request for information on a single category.
func (server *Server) HandleGetCatView(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
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

// HandleGetThread handles a GET request for information on a thread.
func (server *Server) HandleGetThread(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
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

// HandleWritePost handles a POST request to post a new post.
func (server *Server) HandleWritePost(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
	ctx, cancel := context.WithTimeout(req.Context(), time.Second*20)
	defer cancel()

	catName := params.ByName("cat")
	threadNumber, err := strconv.Atoi(params.ByName("thread"))
	if err != nil {
		badRequest("Invalid thread number")(rw, req)
		return
	}

	// If given thread number is not zero, look up OP's unique ID
	parentUID := ""
	if threadNumber != 0 {
		op, err := server.store.GetPostByNumber(ctx, catName, threadNumber)
		if err != nil {
			if errors.Is(err, data.ErrNotFound) {
				notFound("No such thread")(rw, req)
				return
			}
			internalError("Sorry, an error occurred while saving your post")(rw, req)
			return
		}
		if op.IsReply() {
			notFound("No such thread")(rw, req)
			return
		}
		parentUID = op.UID
	}

	// Decode body and write post
	var p data.UserPost
	json.NewDecoder(req.Body).Decode(&p)
	content, errMessage := data.CheckContent(p.Content)
	if len(errMessage) > 0 {
		badRequest(errMessage)(rw, req)
		return
	}

	trans, err := server.store.Trans(ctx)
	if err != nil {
		internalError("Sorry, an error occurred while saving your post")(rw, req)
		log.Println(err)
		return
	}
	err = trans.WritePost(ctx, &data.Post{
		Content:   content,
		Cat:       catName,
		ParentUID: parentUID,
	})
	if err != nil {
		internalError("Sorry, an error occurred while saving your post.")(rw, req)
		log.Println(err)
		return
	}
	trans.Commit(ctx)

	err = json.NewEncoder(rw).Encode(ok{
		Message: "Post submitted",
	})

	if err != nil {
		log.Println(err)
	}
}

// Handle handleCORSPreflight pre-flighting
func handleCORSPreflight(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Methods", "GET,POST")
	rw.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	rw.WriteHeader(http.StatusNoContent)
}

func middlewareCORS(hand httprouter.Handle) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		hand(rw, req, params)
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
	router.GlobalOPTIONS = http.HandlerFunc(handleCORSPreflight)
	router.GET("/v1", middlewareCORS(server.HandleGetCategories))
	router.GET("/v1/:cat", middlewareCORS(server.HandleGetCatView))
	router.POST("/v1/:cat/:thread", middlewareCORS(server.HandleWritePost))
	router.GET("/v1/:cat/:thread", middlewareCORS(server.HandleGetThread))

	server.httpServer.Handler = router
	return server
}
