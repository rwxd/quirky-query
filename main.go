package main

import (
	"flag"
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rwxd/quirky-query/internal"
	"golang.org/x/exp/slog"
	"golang.org/x/net/websocket"
)

var (
	flagVerbose = flag.Bool("v", false, "verbose mode")
	flagPort    = flag.String("port", "8000", "port to run webserver on")
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	flag.Parse()

	logLevel := slog.LevelWarn
	if *flagVerbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)
	slog.Info("Starting", "logLevel", logLevel, "port", *flagPort)

	tracker := internal.NewTracker()
	tracker.CleanUpLoop()

	templates := &Template{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}

	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{}))
	e.Use(middleware.Recover())
	e.Use(tracker.RequestTrackerMiddleware)

	e.Renderer = templates

	e.GET("/", Home)
	e.GET("/stream", func(c echo.Context) error { return Stream(c, tracker) })

	e.Logger.Fatal(e.Start(":" + *flagPort))
}

func Home(c echo.Context) error {
	fqdn := os.Getenv("FQDN")
	if fqdn == "" {
		fqdn = "localhost:" + *flagPort
	}

	ws_secure := true
	if os.Getenv("WS_SECURE") == "" {
		ws_secure = false
	}

	return c.Render(http.StatusOK, "index.html", map[string]interface{}{"fqdn": fqdn, "ws_secure": ws_secure})
}

func Stream(c echo.Context, tracker *internal.Tracker) error {
	slog.Info("Stream Requested", "remoteAddr", c.Request().RemoteAddr)
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		tracker.AddWebsocket(ws)
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}
