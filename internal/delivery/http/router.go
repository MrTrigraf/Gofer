package http

import (
	"net/http"

	"github.com/gofer/internal/delivery/http/middleware"
	"github.com/gofer/internal/infrastructure/postgres"
	"github.com/gofer/internal/infrastructure/websocket"
	"github.com/gofer/internal/usecase/auth"
	"github.com/gofer/internal/usecase/channel"
	"github.com/gofer/internal/usecase/direct"
	"github.com/gofer/internal/usecase/user"
	"github.com/gofer/pkg/config"
	"github.com/gofer/pkg/hasher"
	"github.com/gofer/pkg/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Router struct {
	mux *http.ServeMux
}

func NewRouter(pool *pgxpool.Pool, cfg *config.Config) (*Router, *websocket.Hub) {
	// репозитории
	userRepo := postgres.NewUserRepo(pool)
	channelRepo := postgres.NewChannelRepo(pool)
	messageRepo := postgres.NewMessageRepo(pool)
	directRepo := postgres.NewDirectRepo(pool)

	// usecase'ы
	pwdHasher := hasher.New()
	tokenSvc := jwt.NewTokenService(&cfg.JWT)
	authUC := auth.New(userRepo, pwdHasher, tokenSvc)
	channelUC := channel.New(userRepo, channelRepo, messageRepo)
	directUC := direct.New(userRepo, directRepo, messageRepo)
	userUC := user.New(userRepo)

	// websocket
	hub := websocket.NewHub(channelRepo, messageRepo)
	wsHandler := websocket.NewHandler(hub, tokenSvc)

	// middleware и хендлеры
	jwtMW := middleware.NewJWTMiddleware(tokenSvc)
	authHandler := NewAuthHandler(authUC)
	channelHandler := NewChannelHandler(channelUC)
	directHandler := NewDirectHandler(directUC)
	userHandler := NewUserHandler(userUC)

	r := &Router{mux: http.NewServeMux()}
	r.setupRoutes(jwtMW, authHandler, channelHandler, directHandler, userHandler, wsHandler)
	return r, hub
}

func (r *Router) setupRoutes(
	jwtMW *middleware.JWTMiddleware,
	authHandler *AuthHandler,
	channelHandler *ChannelHandler,
	directHandler *DirectHandler,
	userHandler *UserHandler,
	wsHandler *websocket.Handler,
) {
	// публичные маршруты
	r.mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	r.mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	r.mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)

	// защищённые маршруты
	r.mux.HandleFunc("GET /api/v1/channels", jwtMW.Handle(channelHandler.List))
	r.mux.HandleFunc("POST /api/v1/channels", jwtMW.Handle(channelHandler.Create))
	r.mux.HandleFunc("POST /api/v1/channels/{id}/join", jwtMW.Handle(channelHandler.Join))
	r.mux.HandleFunc("GET /api/v1/channels/{id}/messages", jwtMW.Handle(channelHandler.History))

	r.mux.HandleFunc("GET /api/v1/direct", jwtMW.Handle(directHandler.List))
	r.mux.HandleFunc("POST /api/v1/direct/{user_id}", jwtMW.Handle(directHandler.Start))
	r.mux.HandleFunc("GET /api/v1/direct/{id}/messages", jwtMW.Handle(directHandler.History))

	r.mux.HandleFunc("GET /api/v1/users/search", jwtMW.Handle(userHandler.Search))
	r.mux.HandleFunc("GET /ws", wsHandler.ServeWS)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}
