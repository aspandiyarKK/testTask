package rest

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"testTask/pkg/repository"
)

type App interface {
	Registration(ctx context.Context, user repository.User) (int, error)
	Login(ctx context.Context, user repository.User) (repository.User, error)
}

type Router struct {
	log    *logrus.Entry
	router *gin.Engine
	app    App
	hub    *Hub
}

func NewRouter(log *logrus.Logger, app App) *Router {
	r := &Router{
		log:    log.WithField("component", "router"),
		router: gin.Default(),
		app:    app,
	}
	g := r.router.Group("/api/v1")
	g.POST("/registration", r.registration)
	g.POST("/login", r.login)
	r.hub = newHub()
	go r.hub.run()
	g.Any("/chat", r.wsHandler())
	return r
}

func (r *Router) wsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		func(w http.ResponseWriter, req *http.Request) {
			serveWs(r.hub, w, req)
		}(c.Writer, c.Request)
	}
}

func (r *Router) Run(_ context.Context, addr string) error {
	return r.router.Run(addr)
}

func (r *Router) registration(c *gin.Context) {
	var input repository.User
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	id, err := r.app.Registration(c, input)
	switch {
	case err == nil:
	case errors.Is(err, repository.ErrInvalidCredentials):
		c.JSON(http.StatusForbidden, err.Error())
		return
	case errors.Is(err, repository.ErrAlreadyExists):
		c.JSON(http.StatusConflict, err.Error())
		return
	default:
		r.log.Errorf("failed to register user: %v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (r *Router) login(c *gin.Context) {
	var input repository.User
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	user, err := r.app.Login(c, input)
	switch {
	case err == nil:
	case errors.Is(err, repository.ErrInvalidCredentials):
		c.JSON(http.StatusForbidden, err.Error())
		return
	default:
		r.log.Errorf("failed to login user: %v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, user)
}

//func (r *Router) serveHome(w http.ResponseWriter, req *http.Request) {
//	r.log.Info()
//	if req.URL.Path != "/" {
//		http.Error(w, "Not found", http.StatusNotFound)
//		return
//	}
//	if req.Method != http.MethodGet {
//		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
//		return
//	}
//	http.ServeFile(w, req, "home.html")
//}
