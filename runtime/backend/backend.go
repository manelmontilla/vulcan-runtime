// Package backend defines the interface of a runtime backend that runs Vulcan
// checks. A Backend is responsible for running the container image and the
// parameters of a Check using a concrete engine, for instance: Docker or K8s.
package backend

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
}

// RunResult defines the info returned by the [Backend.Run] function.
type RunResult struct {
	Output []byte
	Error  error
}
