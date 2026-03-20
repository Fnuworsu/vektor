package proxy

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/Fnuworsu/vektor/internal/backend"
	"github.com/Fnuworsu/vektor/internal/events"
	"github.com/Fnuworsu/vektor/internal/proxy/resp"
	"github.com/Fnuworsu/vektor/internal/proxy/router"
)

type Server struct {
	addr     string
	store    backend.BackendStore
	eventCh  chan<- events.AccessEvent
	listener net.Listener
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewServer(addr string, store backend.BackendStore, eventCh chan<- events.AccessEvent) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		addr:    addr,
		store:   store,
		eventCh: eventCh,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (s *Server) Start() error {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = l

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				log.Printf("accept error: %v", err)
				continue
			}
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}()

	return nil
}

func (s *Server) Stop() {
	s.cancel()
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	reader := bufio.NewReader(conn)
	routerInst := router.NewRouter(s.store, s.eventCh)
	clientAddr := conn.RemoteAddr().String()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		cmd, err := resp.ParseCommand(reader)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		conn.SetReadDeadline(time.Time{})

		err = routerInst.Dispatch(s.ctx, cmd, conn, clientAddr)
		if err != nil {
			if err == router.ErrQuit {
				return
			}
			log.Printf("dispatch error: %v", err)
			return
		}
	}
}
