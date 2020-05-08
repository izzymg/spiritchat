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
func (server *Server) HandleGetCategories(ctx context.Context, req *request, respond respondFunc) {
	categories, err := server.store.GetCategories(ctx)
	if err != nil {
		respond(
			http.StatusInternalServerError,
			nil, "Sorry, an error occurred while fetching categories",
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
				http.StatusInternalServerError,
				nil, err.Error(),
			)
			return
		}
		respond(
			http.StatusInternalServerError,
			nil, "Sorry, an error occurred while fetching the category's threads",
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
		respond(http.StatusNotFound, nil, "Invalid thread number")
		return
	}
	threadView, err := server.store.GetThreadView(ctx, req.params.ByName("cat"), threadNum)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			respond(http.StatusNotFound, nil, err.Error())
			return
		}
		respond(http.StatusInternalServerError, nil, "Sorry, an error occurred while fetching the thread")
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

	// If given thread number is not zero, look up OP's unique ID
	parentUID := ""
	if threadNumber != 0 {
		op, err := server.store.GetPostByNumber(ctx, catName, threadNumber)
		if err != nil {
			if errors.Is(err, data.ErrNotFound) {
				respond(
					http.StatusNotFound,
					nil, "Invalid thread number",
				)
				return
			}
			respond(
				http.StatusInternalServerError,
				nil, "Sorry, an error occurred while saving your post",
			)
			return
		}
		if op.IsReply() {
			respond(
				http.StatusNotFound,
				nil, "No such found",
			)
			return
		}
		parentUID = op.UID
	}

	// Decode body and write post
	var p data.UserPost
	json.NewDecoder(req.rawRequest.Body).Decode(&p)
	content, errMessage := data.CheckContent(p.Content)
	if len(errMessage) > 0 {
		respond(
			http.StatusBadRequest,
			nil, errMessage,
		)
		return
	}

	trans, err := server.store.Trans(ctx)
	rollback := func() {
		err := trans.Rollback(ctx)
		if err != nil {
			log.Printf("Failed to rollback transaction: %s", err)
		}
	}
	if err != nil {
		rollback()
		if err != nil {
			log.Printf("Failed to rollback transaction: %s", err)
		}
		respond(
			http.StatusInternalServerError,
			nil, "Sorry, an error occurred while saving your post",
		)
		log.Printf("Failed to create post write transaction: %s", err)
		return
	}
	err = trans.WritePost(ctx, &data.Post{
		Content:   content,
		Cat:       catName,
		ParentUID: parentUID,
	})
	if err != nil {
		rollback()
		if errors.Is(err, data.ErrNotFound) {
			respond(http.StatusBadRequest, nil, err.Error())
			return
		}
		respond(
			http.StatusInternalServerError,
			nil, "Sorry, an error occurred while saving your post",
		)
		log.Printf("Failed to write post to db: %s", err)
		return
	}
	err = trans.Commit(ctx)
	if err != nil {
		rollback()
		log.Printf("Failed to commit post write: %s", err)
	}

	respond(http.StatusOK, ok{Message: "Post submitted"}, "")
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
	router.GET("/v1", middlewareCORS(genHandler(server.HandleGetCategories)))
	router.GET("/v1/:cat", middlewareCORS(genHandler(server.HandleGetCatView)))
	router.POST("/v1/:cat/:thread", middlewareCORS(genHandler(server.HandleWritePost)))
	router.GET("/v1/:cat/:thread", middlewareCORS(genHandler(server.HandleGetThreadView)))

	server.httpServer.Handler = router
	return server
}
