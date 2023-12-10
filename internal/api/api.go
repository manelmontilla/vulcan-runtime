// Package api implements a generic HTTP API called by the checks to push their
// status and results.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"path"
	"sync"

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
	srv    http.Server
}

func NewPush(addr string) *Push {
	log := slog.Default()
	p := &Push{
		log: log,
	}
	m := http.NewServeMux()
	srv := http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(p.handleHTTP),
	}

	return p
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
	if rp != "/checks/" {
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

func (c *Push) start() {

}

func writeHTTPError(status int, msg string, w http.ResponseWriter) {
	w.WriteHeader(status)
	if msg != "" {
		w.Write([]byte(msg))
	}
}
