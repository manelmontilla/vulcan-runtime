// Package api implements a generic HTTP API called by the checks to push their
// status and results.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/manelmontilla/vulcan-runtime/runtime"
)

// State defines the payload sent by the check when publishing their status.
type State struct {
	ID       string         `json:"id" validate:"required"`
	Status   *runtime.State `json:"status,omitempty"`
	AgentID  *string        `json:"agent_id,omitempty"`
	Report   *string        `json:"report,omitempty"`
	Raw      *string        `json:"raw,omitempty"`
	Progress *float32       `json:"progress,omitempty"`
}

// Push implements the REST Push used called by the checks to communicate its
// progress and results.
type Push struct {
	checks sync.Map
	log    *slog.Logger
}

// NewPush creates a new HTTP server that listens for check progress
// notifications.
func NewPush(addr string) *Push {
	log := slog.Default()
	p := &Push{
		log: log,
	}
	return p
}

// Start makes the Push API start listening for notifications. It will try to
// gracefully stop listening when the passed context is cancelled. The returnned
// channel will contain the result of the stop operation.
func (p *Push) Start(ctx context.Context, addr string) <-chan error {
	srv := &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(p.handleHTTP),
	}
	stopped := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		stopped <- err
	}()

	done := make(chan error)
	ctxDone := ctx.Done()
	go func() {
		var err error
	Loop:
		for {
			select {
			case <-ctxDone:
				ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				err = srv.Shutdown(ctxTimeout)
				// After the ctxDone channel was fired we don't need to consider
				// it anymore in the select.
				ctxDone = nil
			case stoppedErr := <-stopped:
				// This select branch waits for the HTTP server to be stopped.
				if errors.Is(stoppedErr, http.ErrServerClosed) {
					break Loop
				}
				if stoppedErr != nil {
					err = stoppedErr
				}
				break Loop
			}
		}
		done <- err
	}()
	return stopped
}

func (p *Push) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Checks pushes the information using requests of type: PATCH /check/$id.

	if r.Method != http.MethodPatch {
		p.log.Error("unable to process check push notification, invalid method", "method", r.Method)
		writeHTTPError(http.StatusBadRequest, "invalid method", w)
		return
	}

	// URL cannot be nil.
	rp := r.URL.Path
	dir, id := path.Split(rp)
	if dir != "/checks/" {
		p.log.Error("unable to process check push notification, invalid path", "path", rp)
		writeHTTPError(http.StatusBadRequest, "invalid path", w)
		return
	}
	progress, ok := p.checks.Load(id)
	if !ok {
		p.log.Error("unable to process check push notification, check id not found", "id", id)
		writeHTTPError(http.StatusBadRequest, "check id not found", w)
		return
	}

	dec := json.NewDecoder(r.Body)
	var s State
	if err := dec.Decode(&s); err != nil {
		p.log.Error("unable to process check push notification, unable to parse body", "err", err)
		writeHTTPError(http.StatusBadRequest, "invalid body", w)
		return
	}

	cprogress, ok := progress.(chan<- State)
	if !ok {
		p.log.Error("unable to process check push notification, unexpected channel type")
		writeHTTPError(http.StatusInternalServerError, "", w)
		return
	}

	cprogress <- s
	w.WriteHeader(http.StatusOK)
}

func writeHTTPError(status int, msg string, w http.ResponseWriter) {
	w.WriteHeader(status)
	if msg != "" {
		w.Write([]byte(msg))
	}
}
