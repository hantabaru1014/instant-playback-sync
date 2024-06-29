package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/xid"
	slogecho "github.com/samber/slog-echo"
)

type Server struct {
	Config *Config

	upgrader *websocket.Upgrader
	roomMap  roomMap
}

func NewServer(config *Config) *Server {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	return &Server{
		Config:   config,
		upgrader: upgrader,
		roomMap:  *newRoomMap(),
	}
}

func (s *Server) Run(address string) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(slogecho.NewWithFilters(
		slog.Default(),
		slogecho.IgnorePath("/live"), // k8s liveness and readiness probes
	))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet},
	}))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
			return next(c)
		}
	})

	// アセット配信
	e.GET("/", func(c echo.Context) error {
		return c.File("front/index.html")
	})
	e.GET("/r/:id", func(c echo.Context) error {
		return c.File("front/room.html")
	})
	e.GET("/i.js", func(c echo.Context) error {
		return c.File("front/inject.js")
	})
	e.GET("/wsproxy", func(c echo.Context) error {
		return c.File("front/wsproxy.html")
	})

	// API
	e.GET("/api/rooms/:id/ws", s.handleWSRequest)
	e.GET("/api/rooms/:id", s.handleGetRooms)

	// k8s liveness and readiness probes
	e.GET("/live", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		if err := e.Start(address); err != nil && err != http.ErrServerClosed {
			slog.Error("Fatal error on run server", err)
		}
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	slog.Info("Signal interrupt. Shutting down server...")
	s.shutdownRooms()
	if err := e.Shutdown(ctx); err != nil {
		slog.Error("Fatal error on shutdown server", err)
	}
}

func (s *Server) shutdownRooms() {
	s.roomMap.each(func(r *Room) {
		r.Shutdown(websocket.FormatCloseMessage(1001, "Server is shutting down"))
	})
}

func (s *Server) handleWSRequest(c echo.Context) error {
	id := c.Param("id")
	if id == "" || len(id) > 32 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid room id"})
	}

	respWriter := c.Response().Writer
	conn, err := s.upgrader.Upgrade(respWriter, c.Request(), respWriter.Header())
	if err != nil {
		return err
	}

	r := s.roomMap.get(id)
	if r == nil {
		r = NewRoom(id, s)
		s.roomMap.add(r)
		go r.run()
	}

	req := c.Request()
	sessionId := xid.New().String()
	reqLog := map[string]interface{}{
		"path":    req.URL.Path,
		"query":   req.URL.Query(),
		"ip":      c.RealIP(),
		"referer": req.Referer(),
	}
	slog.Info("New WSRequest", "room_id", id, "session_id", sessionId, "request", reqLog)

	err = r.handleRequest(conn, sessionId)

	func() {
		rlen := r.Len()
		if rlen > 1 {
			return
		}
		if rlen == 1 {
			if ss := r.all(); len(ss) != 1 || ss[0].ID != sessionId {
				return
			}
		}
		s.roomMap.del(id)
		r.Shutdown([]byte{})
		slog.Info("Room is empty. so deleted.", "room_id", id)
	}()

	return err
}

func (s *Server) handleGetRooms(c echo.Context) error {
	id := c.Param("id")
	r := s.roomMap.get(id)
	if r == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "room not found"})
	}
	d := r.ToDTO()
	return c.JSON(http.StatusOK, d)
}
