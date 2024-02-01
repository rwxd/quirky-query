package internal

import (
	"io"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slog"
	"golang.org/x/net/websocket"
)

var (
	IgnoredPaths = []string{
		"/favicon.ico",
		"/robots.txt",
	}
)

type TrackerQueueItem struct {
	Path   string
	Method string
	Query  string
	Body   string
}

func (t *TrackerQueueItem) String() string {
	return t.Method + " " + t.Path + " " + t.Query + " " + t.Body
}

type Tracker struct {
	Stream  chan TrackerQueueItem
	Clients map[*websocket.Conn]bool
	sync.Mutex
}

func (s *Tracker) RequestTrackerMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := next(c); err != nil {
			c.Error(err)
		}

		for _, path := range IgnoredPaths {
			if path == c.Request().URL.Path {
				return nil
			}
		}

		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}

		item := TrackerQueueItem{
			Path:   c.Request().URL.Path,
			Method: c.Request().Method,
			Query:  c.Request().URL.RawQuery,
			Body:   string(bodyBytes),
		}

		s.Stream <- item
		return nil
	}
}

func (s *Tracker) AddWebsocket(ws *websocket.Conn) {
	slog.Debug("Adding websocket to stream", "client", ws.Request().RemoteAddr)
	s.Lock()
	s.Clients[ws] = true
	s.Unlock()

	defer func() {
		s.Lock()
		delete(s.Clients, ws)
		s.Unlock()
		slog.Debug("Removing websocket from stream", "client", ws.RemoteAddr())
		ws.Close()
	}()

	for item := range s.Stream {
		s.Lock()
		for client := range s.Clients {
			safeString, err := inputToSafeHTML(item.String())
			if err != nil {
				slog.Error("Could not convert string to safe HTML", "error", err)
				break
			}

			err = websocket.Message.Send(client, safeString)
			if err != nil {
				slog.Error("Could not send message to websocket", "error", err)
				client.Close()
				delete(s.Clients, client)
			}
		}
		s.Unlock()
	}
}

// Runs in a goroutine to clean up the stream channel
// If the channel is full we will drop the oldest item
// If the channel was not accessed for a while we will drop all items
func (s *Tracker) CleanUpLoop() {
	go func() {
		for {
			if len(s.Stream) == cap(s.Stream) {
				slog.Debug("Stream is full, dropping oldest item")
				<-s.Stream
			}
			time.Sleep(1 * time.Millisecond)
		}
	}()

	go func() {
		lastLength := 0
		for {
			if len(s.Stream) == lastLength && lastLength > 0 && lastLength != cap(s.Stream) {
				slog.Debug("Stream is not accessed, dropping all items")
				for range s.Stream {
					<-s.Stream
				}
			}
			lastLength = len(s.Stream)
			time.Sleep(5 * time.Second)
		}
	}()
}

func NewTracker() *Tracker {
	return &Tracker{
		Stream:  make(chan TrackerQueueItem, 10000),
		Clients: make(map[*websocket.Conn]bool),
	}
}
