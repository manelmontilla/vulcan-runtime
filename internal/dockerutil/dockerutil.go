// Copyright 2023 Adevinta

// Package dockerutil provides Docker utility functions.
package dockerutil

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
)

// NewAPIClient returns a new Docker API client. This client behaves
// as close as possible to the Docker CLI. It gets its configuration
// from the Docker config file and honors the [Docker CLI environment
// variables]. It also sets up TLS authentication if TLS is enabled.
//
// [Docker CLI environment variables]: https://docs.docker.com/engine/reference/commandline/cli/#environment-variables
func NewAPIClient() (client.APIClient, error) {
	tlsVerify := os.Getenv(client.EnvTLSVerify) != ""

	var tlsopts *tlsconfig.Options
	if tlsVerify {
		certPath := os.Getenv(client.EnvOverrideCertPath)
		if certPath == "" {
			certPath = config.Dir()
		}
		tlsopts = &tlsconfig.Options{
			CAFile:   filepath.Join(certPath, flags.DefaultCaFile),
			CertFile: filepath.Join(certPath, flags.DefaultCertFile),
			KeyFile:  filepath.Join(certPath, flags.DefaultKeyFile),
		}
	}

	opts := &flags.ClientOptions{
		TLS:        tlsVerify,
		TLSVerify:  tlsVerify,
		TLSOptions: tlsopts,
	}

	return command.NewAPIClientFromFlags(opts, config.LoadDefaultConfigFile(io.Discard))
}

// ImageLabels returns the labels defined in an image.
func ImageLabels(cli client.APIClient, image string) (map[string]string, error) {
	ctx := context.Background()
	filter := filters.KeyValuePair{
		Key:   "reference",
		Value: image,
	}
	options := types.ImageListOptions{
		Filters: filters.NewArgs(filter),
	}
	infos, err := cli.ImageList(ctx, options)
	if err != nil {
		return nil, err
	}
	var labels = make(map[string]string)
	for _, info := range infos {
		for k, v := range info.Labels {
			labels[k] = v
		}
	}
	return labels, nil
}

// ParseImageRef validates and enrich an image reference with domain (docker.io if
// domain missing), tag (latest if missing).
func ParseImageRef(ref string) (domain, path, tag string, err error) {
	named, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return
	}
	// add latests if tag is missing
	named = reference.TagNameOnly(named)
	domain = reference.Domain(named)
	path = reference.Path(named)
	tagged, isTagged := named.(reference.Tagged)
	if domain == "docker.io" {
		// Ignore docker.io and "library/" as we don't expect a check in
		// docker.io/library.
		path = strings.TrimPrefix(path, "library/")
	}
	if isTagged {
		tag = tagged.Tag()
	}
	return
}
