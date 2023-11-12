package checktype

import (
	"errors"
	"fmt"

	catalog "github.com/adevinta/vulcan-check-catalog/pkg/model"

	"github.com/manelmontilla/vulcan-runtime/internal/dockerutil"
)

// Checktype defines the information of a Checktype.
type Checktype struct {
	catalog.Checktype
	Version string
}

// FromImageRef returns the information of an checktype given the reference
// to to its container image.
func FromImageRef(ref string) (Checktype, error) {
	// Try to gather information from the container image config data.
	image, err := imageFromRef(ref)
	if errors.Is(err, ErrNoChecktypeImage{}) {
		// If the image doesn't contain the info for the checktype we need to
		// fallback to only gathering the info encoded in the URI i,e: the name
		// and the version of the checktype.
		return fromRefURI(ref)
	}
	if err != nil {
		return Checktype{}, fmt.Errorf("unable to get Checktype info from ref %s: %v", ref, err)
	}
	ct, err := image.Checktype()
	if err != nil {
		return Checktype{}, fmt.Errorf("unable to get Checktype info from ref %s: %v", ref, err)
	}
	_, _, tag, err := dockerutil.ParseImageRef(ref)
	if err != nil {
		err = fmt.Errorf("unable to parse image ref %s: %w", ref, err)
		return Checktype{}, err
	}
	return Checktype{
		Checktype: ct,
		Version:   tag,
	}, nil
}

// fromRefURI extracts checktype data from a container image URI.
func fromRefURI(ref string) (Checktype, error) {
	domain, path, tag, err := dockerutil.ParseImageRef(ref)
	if err != nil {
		err = fmt.Errorf("unable to parse image ref %s: %w", ref, err)
		return Checktype{}, err
	}
	name := fmt.Sprintf("%s/%s", domain, path)
	if domain == "docker.io" {
		name = path
	}
	version := tag
	return Checktype{
		Checktype: catalog.Checktype{
			Name: name,
		},
		Version: version,
	}, nil
}
