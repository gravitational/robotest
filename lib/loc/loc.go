package loc

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gravitational/trace"
)

func NewLocator(repository, name, version string) *Locator {
	return &Locator{Repository: repository, Name: name, Version: version}
}

func MustParseLocator(locatorS string) Locator {
	locator, err := ParseLocator(locatorS)
	if err != nil {
		panic(err)
	}
	return *locator
}

func ParseLocator(locatorS string) (*Locator, error) {
	parts := strings.Split(locatorS, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, trace.BadParameter(
			"invalid package locator format: expected repository/name:semver, got %q", locatorS)
	}
	match := reLocator.FindAllStringSubmatch(parts[1], -1)
	if len(match) != 1 || len(match[0]) != 3 {
		return nil, trace.BadParameter(
			"invalid package locator format: expected repository/name:semver, got %q", locatorS)
	}
	return NewLocator(parts[0], match[0][1], match[0][2]), nil
}

// Locator defines a package identificator.
// Locator consists of a repository name, package name and a version (in SemVer format)
type Locator struct {
	Repository string `json:"repository"`
	Name       string `json:"name"`
	Version    string `json:"version"`
}

func (r Locator) String() string {
	return fmt.Sprintf("%v/%v:%v", r.Repository, r.Name, r.Version)
}

func (r Locator) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

func (r *Locator) UnmarshalText(p []byte) error {
	locator, err := ParseLocator(string(p))
	if err != nil {
		return err
	}
	if r == nil {
		r = &Locator{}
	}
	*r = *locator
	return nil
}

// reLocator defines a regular expression to recognize a package locator format
var reLocator = regexp.MustCompile(`^([a-zA-Z0-9\-_\.]+):(.+)$`)
