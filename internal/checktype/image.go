package checktype

import (
	"fmt"

	catalog "github.com/adevinta/vulcan-check-catalog/pkg/model"

	"github.com/manelmontilla/vulcan-runtime/internal/dockerutil"
)

const (
	// checktypeNameLabel defines the key of the label using [reverse DNS notation].
	//
	// [reverse DNS notation]:https://docs.docker.com/config/labels-custom-metadata/
	checktypeNameLabel = "com.adevinta.vulcan.name"

	// checktypeManifest defines the key of the label using [reverse DNS notation].
	//
	// [reverse DNS notation]:https://docs.docker.com/config/labels-custom-metadata/
	checktypeManifest = "com.adevinta.vulcan.manifest"
)

// ErrNoChecktypeImage is returned by the [ImageFromRef] function when an image
// does not contain the metadata of a checktype.
type ErrNoChecktypeImage struct {
	Image string
}

func (e ErrNoChecktypeImage) Error() string {
	return fmt.Sprintf("invalid metadata in image %s", e.Image)
}

// image represents the metadata about a checktype stored in a docker image.
// Vulcan checktype.
type image struct {
	// Name the name of the image in format REPOSITORY:TAG.
	Name string

	// ChecktypeName the name of the checktype that the image contains.
	ChecktypeName string

	// Manifest the manifest of the checktype that the image contains.
	Manifest Manifest
}

// imageFromRef returns the information of a checktype stored in the image
// pointed by a ref.
func imageFromRef(ref string) (image, error) {
	cli, err := dockerutil.NewAPIClient()
	if err != nil {
		return image{}, fmt.Errorf("unable to instantiate a docker client: %v", err)
	}
	labels, err := dockerutil.ImageLabels(cli, ref)
	if err != nil {
		return image{}, fmt.Errorf("unable to read image labels: %w", err)
	}

	ctName, ok := labels[checktypeNameLabel]
	if !ok {
		err := ErrNoChecktypeImage{Image: ref}
		return image{}, fmt.Errorf("label %s not found: %w", checktypeNameLabel, err)
	}

	m, ok := labels[checktypeManifest]
	if !ok {
		err := ErrNoChecktypeImage{Image: ref}
		return image{}, fmt.Errorf("label %s not found: %w", checktypeManifest, err)
	}

	manifest, err := parseManifest(m)
	if err != nil {
		err := ErrNoChecktypeImage{Image: ref}
		return image{}, fmt.Errorf("invalid checktype manifest: %w", err)
	}

	return image{
		Name:          ref,
		ChecktypeName: ctName,
		Manifest:      manifest,
	}, nil
}

// Checktype returns the information of the checktype defined in the image.
func (i image) Checktype() (catalog.Checktype, error) {
	options, err := i.Manifest.UnmarshalOptions()
	if err != nil {
		return catalog.Checktype{}, fmt.Errorf("unable to unmarshal options: %w", err)
	}
	assetTypes, err := i.Manifest.AssetTypes.Strings()
	if err != nil {
		return catalog.Checktype{}, fmt.Errorf("unable to read asset types: %w", err)
	}
	var requiredVars []any
	for _, r := range i.Manifest.RequiredVars {
		requiredVars = append(requiredVars, r)
	}
	ct := catalog.Checktype{
		Name:         i.ChecktypeName,
		Description:  i.Manifest.Description,
		Image:        i.Name,
		Timeout:      i.Manifest.Timeout,
		Options:      options,
		RequiredVars: requiredVars,
		Assets:       assetTypes,
	}
	return ct, nil
}
