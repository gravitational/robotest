/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gce

import (
	"fmt"
	"hash/fnv"
	"io"
)

// TranslateClusterName translates the specified cluster name
// to comply with Google Cloud Platform naming convention
// See: https://cloud.google.com/compute/docs/labeling-resources
func TranslateClusterName(cluster string) string {
	// Use a hash to fit the resource name restriction on GCE subject to RFC1035
	return fmt.Sprintf("robotest-%x", Hash(cluster))
}

// Hash computes a hash from the given set of strings
func Hash(strings ...string) uint32 {
	digester := fnv.New32()
	for _, s := range strings {
		_, _ = io.WriteString(digester, s)
	}
	return digester.Sum32()
}
