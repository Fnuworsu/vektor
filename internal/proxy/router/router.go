package router

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/Fnuworsu/vektor/internal/backend"
	"github.com/Fnuworsu/vektor/internal/events"
	"github.com/Fnuworsu/vektor/internal/proxy/resp"
)

var ErrQuit = errors.New("client quit")

type Router struct {
	store   backend.BackendStore
	eventCh chan<- events.AccessEvent
}

func NewRouter(store backend.BackendStore, eventCh chan<- events.AccessEvent) *Router {
	return &Router{
		store:   store,
		eventCh: eventCh,
	}
}

func (r *Router) Dispatch(ctx context.Context, cmd []string, w io.Writer, clientAddr string) error {
	if len(cmd) == 0 {
		return resp.WriteError(w, errors.New("empty command"))
	}

	op := strings.ToUpper(cmd[0])

	switch op {
	case "GET":
		if len(cmd) != 2 {
			return resp.WriteError(w, errors.New("ERR wrong number of arguments for 'get' command"))
		}
		key := cmd[1]

		select {
		case r.eventCh <- events.AccessEvent{
			Key:        key,
			OccurredAt: time.Now(),
			ClientAddr: clientAddr,
		}:
		default:
		}

		val, err := r.store.Get(ctx, key)
		if err != nil {
			return resp.WriteError(w, err)
		}
		return resp.WriteBulkString(w, val)

	case "SET":
		if len(cmd) < 3 {
			return resp.WriteError(w, errors.New("ERR wrong number of arguments for 'set' command"))
		}
		err := r.store.Set(ctx, cmd[1], []byte(cmd[2]))
		if err != nil {
			return resp.WriteError(w, err)
		}
		return resp.WriteSimpleString(w, "OK")

	case "DEL":
		if len(cmd) < 2 {
			return resp.WriteError(w, errors.New("ERR wrong number of arguments for 'del' command"))
		}
		err := r.store.Delete(ctx, cmd[1])
		if err != nil {
			return resp.WriteError(w, err)
		}
		return resp.WriteInteger(w, 1)

	case "PING":
		err := r.store.Ping(ctx)
		if err != nil {
			return resp.WriteError(w, err)
		}
		return resp.WriteSimpleString(w, "PONG")

	case "QUIT":
		resp.WriteSimpleString(w, "OK")
		return ErrQuit

	default:
		return resp.WriteError(w, errors.New("COMMAND_NOT_SUPPORTED"))
	}
}
