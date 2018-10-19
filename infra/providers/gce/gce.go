package gce

import "strings"

// TranslateClusterName translates the specified cluster name
// to comply with Google Cloud Platform naming convention
// See: https://cloud.google.com/compute/docs/labeling-resources
func TranslateClusterName(cluster string) string {
	return strings.ToLower(replacer.Replace(cluster))
}

var replacer = strings.NewReplacer(".", "-")
