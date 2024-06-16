package app

import (
	"log/slog"
	"net/http"

	"github.com/hantabaru1014/instant-playback-sync/dto"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/olahol/melody"
	slogecho "github.com/samber/slog-echo"
)

type Server struct {
	m              *melody.Melody
	roomMap        roomMap
	isShuttingDown bool
}

func NewServer() *Server {
	return &Server{
		roomMap: *newRoomMap(),
	}
}

func (s *Server) makeMelody() {
	m := melody.New()

	m.HandleConnect(func(ss *melody.Session) {
		slog.Info("WS Client Connected", "address", ss.Request.RemoteAddr)
		if ar, exists := ss.Get(ROOM_KEY); exists {
			r := ar.(*Room)
			r.handleConnect(ss)
		}
	})
	m.HandleDisconnect(func(ss *melody.Session) {
		if ar, exists := ss.Get(ROOM_KEY); exists {
			r := ar.(*Room)
			if r.isEmptyOrErr() {
				s.roomMap.del(r.ID)
				slog.Info("Room is empty. so deleted.", "room_id", r.ID)
			}
		}
		slog.Info("WS Client Disconnected", "address", ss.Request.RemoteAddr)
	})
	m.HandleMessage(func(ss *melody.Session, msg []byte) {
		cmd, err := dto.UnmarshalCmdMsg(msg)
		if err != nil {
			return
		}
		if ar, exists := ss.Get(ROOM_KEY); exists {
			r := ar.(*Room)
			r.handleMessage(cmd, msg, ss)
		}
	})
	s.m = m
}

func (s *Server) Run(address string) {
	e := echo.New()
	s.makeMelody()

	e.Use(slogecho.NewWithFilters(
		slog.Default(),
		slogecho.IgnorePath("/live", "/ready"), // k8s liveness and readiness probes
	))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet},
	}))

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

	// API
	e.GET("/api/rooms/:id/ws", s.handleWSRequest)
	e.GET("/api/rooms/:id", s.handleGetRooms)

	// k8s liveness and readiness probes
	e.GET("/live", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/ready", func(c echo.Context) error {
		if s.isShuttingDown {
			return c.String(http.StatusServiceUnavailable, "Shutting down")
		} else {
			return c.String(http.StatusOK, "OK")
		}
	})

	slog.Error("Fatal error on run server", e.Start(address))
}

func (s *Server) handleWSRequest(c echo.Context) error {
	id := c.Param("id")
	r := s.roomMap.getOrAdd(id, s.m)
	keys := map[string]interface{}{
		ROOM_KEY: r,
	}
	req := c.Request()
	reqLog := map[string]interface{}{
		"path":    req.URL.Path,
		"query":   req.URL.Query(),
		"ip":      req.RemoteAddr,
		"referer": req.Referer(),
	}
	slog.Info("handleWSRequest", "room_id", id, "request", reqLog)
	s.m.HandleRequestWithKeys(c.Response().Writer, c.Request(), keys)
	return nil
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
