package main

import (
	"flag"
	"io"
	"net/http"
	"os"
	"text/template"

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
	return c.Render(http.StatusOK, "index.html", nil)
}

func Stream(c echo.Context, tracker *internal.Tracker) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		tracker.AddWebsocket(ws)
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}
