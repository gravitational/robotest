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

package gravity

import (
	"bufio"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	sshutils "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"
)

// parse `gravity status`
func parseStatus(status *GravityStatus) sshutils.OutputParseFn {
	return func(r *bufio.Reader) error {
		decoder := json.NewDecoder(r)
		return trace.Wrap(decoder.Decode(status))
	}
}

// from https://github.com/gravitational/gravity/blob/master/lib/utils/parse.go
//
// ParseDDOutput parses the output of "dd" command and returns the reported
// speed in bytes per second.
//
// Example output:
//
// $ dd if=/dev/zero of=/tmp/testfile bs=1G count=1
// 1+0 records in
// 1+0 records out
// 1073741824 bytes (1.1 GB) copied, 4.52455 s, 237 MB/s
func ParseDDOutput(output string) (speedBytesPerSec uint64, err error) {
	// 1073741824 bytes (1.1 GB) copied, 4.52455 s, 237 MB/s
	// 1073741824 bytes (1,1 GB, 1,0 GiB) copied, 4,53701 s, 237 MB/s
	output = strings.TrimSpace(output)
	match := speedRe.FindStringSubmatch(output)
	if len(match) != 2 {
		return 0, trace.BadParameter("failed to match speed value (e.g. 237 MB/s) in %q", output)
	}

	// Support comma-formatted floats - depending on selected locale
	speedValue := strings.TrimSpace(strings.Replace(match[1], ",", ".", 1))
	value, err := strconv.ParseFloat(speedValue, 64)
	if err != nil {
		return 0, trace.Wrap(err, "failed to parse speed value as a float: %q", speedValue)
	}

	units := strings.TrimSpace(strings.TrimPrefix(match[0], match[1]))
	switch units {
	case "kB/s":
		return uint64(value * 1000), nil
	case "MB/s":
		return uint64(value * 1000 * 1000), nil
	case "GB/s":
		return uint64(value * 1000 * 1000 * 1000), nil
	default:
		return 0, trace.BadParameter("expected units (one of kB/s, MB/s, GB/s) but got %q", units)
	}
}

var speedRe = regexp.MustCompile(`(\d+(?:[.,]\d+)?) \w+/s$`)
