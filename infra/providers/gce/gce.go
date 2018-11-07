package gce

import (
	"fmt"
	"hash/fnv"
	"io"
	"strings"
)

// TranslateClusterName translates the specified cluster name
// to comply with Google Cloud Platform naming convention
// See: https://cloud.google.com/compute/docs/labeling-resources
func TranslateClusterName(cluster string) string {
	// Use a hash to fit the resource name restriction on GCE subject to RFC1035
	return fmt.Sprintf("robotest-%v", Hash(cluster))
}

// Hash computes a hash from the given set of strings
func Hash(strings ...string) string {
	digester := fnv.New32()
	for _, s := range strings {
		io.WriteString(digester, s)
	}
	return string(digester.Sum(nil))
}

var replacer = strings.NewReplacer(".", "-")
