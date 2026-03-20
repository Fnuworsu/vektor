package cgobridge

/*
#cgo CFLAGS: -I${SRCDIR}/../../engine/include
#cgo LDFLAGS: -L${SRCDIR}/../../engine/build -Wl,-rpath,${SRCDIR}/../../engine/build -lvektor_engine -lstdc++
#include "engine.h"
#include <stdlib.h>

extern void prefetchCallbackCGO(char* key, double prob, void* userdata);

static void proxy_callback_wrapper(const char* key, double prob, void* userdata) {
    prefetchCallbackCGO((char*)key, prob, userdata);
}

static inline void set_proxy_callback(vektor_engine_t* engine, void* userdata) {
    vektor_engine_set_callback(engine, proxy_callback_wrapper, userdata);
}
*/
import "C"

import (
	"errors"
	"sync"
	"time"
	"unsafe"
)

var (
	engineMap sync.Map
	nextID    uintptr = 1
	idMu      sync.Mutex
)

type cgoEngine struct {
	id      uintptr
	engine  *C.vektor_engine_t
	candCh  chan PrefetchCandidate
}

func NewEngine(markovOrder int, maxKeys int, threshold float64) Engine {
	cEngine := C.vektor_engine_create(C.int(markovOrder), C.int(maxKeys), C.double(threshold))

	idMu.Lock()
	id := nextID
	nextID++
	idMu.Unlock()

	e := &cgoEngine{
		id:     id,
		engine: cEngine,
		candCh: make(chan PrefetchCandidate, 100000),
	}

	engineMap.Store(id, e)
	C.set_proxy_callback(cEngine, unsafe.Pointer(id))

	return e
}

//export prefetchCallbackCGO
func prefetchCallbackCGO(key *C.char, prob C.double, userdata unsafe.Pointer) {
	id := uintptr(userdata)
	val, ok := engineMap.Load(id)
	if !ok {
		return
	}
	e := val.(*cgoEngine)

	select {
	case e.candCh <- PrefetchCandidate{
		Key:         C.GoString(key),
		Probability: float64(prob),
	}:
	default:
	}
}

func (e *cgoEngine) Start() {
	C.vektor_engine_start(e.engine)
}

func (e *cgoEngine) Stop() {
	C.vektor_engine_stop(e.engine)
	C.vektor_engine_destroy(e.engine)
	engineMap.Delete(e.id)
	close(e.candCh)
}

func (e *cgoEngine) PushEvent(key string, ts time.Time) error {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	res := C.vektor_engine_push_event(e.engine, cKey, C.int64_t(ts.UnixNano()))
	if res != 0 {
		return errors.New("engine ring buffer is full")
	}
	return nil
}

func (e *cgoEngine) Candidates() <-chan PrefetchCandidate {
	return e.candCh
}

func (e *cgoEngine) GetModelState() uint64 {
	return uint64(C.vektor_engine_get_tracked_keys(e.engine))
}
