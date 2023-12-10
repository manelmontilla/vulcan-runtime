// Package backend defines the interface of a runtime backend that runs Vulcan
// checks. A Backend is responsible for running the container image and the
// parameters of a Check using a concrete engine, for instance: Docker or K8s.
package backend

import (
	"context"
	"net"
)

// RunParams defines the parameters needed by the [Backend.Run] function to run
// a Check.
type RunParams struct {
	CheckID          string
	CheckTypeName    string
	ChecktypeVersion string
	Image            string
	Target           string
	AssetType        string
	Options          string
	RequiredVars     []string
	Metadata         map[string]string
	RuntimeAddr      net.Addr
}

// RunResult defines the info returned by the [Backend.Run] function.
type RunResult struct {
	Output []byte
	Error  error
}

// Backend runs checks using a specific container technology. The backend is
// responsible for pulling the container image of the check, setting its
// environment variables, making available to the proper network the REST API
// the check needs to update its state, and running the container.
type Backend interface {
	Run(ctx context.Context, params RunParams) (<-chan RunResult, error)
}
