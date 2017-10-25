package gravity

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"

	sshutils "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"
)

// i.e. "Status: active"
var rStatusKV = regexp.MustCompile(`^(?P<key>[\w\s]+)\:\s*(?P<val>[\w\d\_\-]+),*.*`)
var rStatusNodeIp = regexp.MustCompile(`^[\s\w\-\d]+\((?P<ip>[\d\.]+)\).*`)

// parse `gravity status`
func parseStatus(status *GravityStatus) sshutils.OutputParseFn {
	return func(r *bufio.Reader) error {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			vars := rStatusKV.FindStringSubmatch(line)
			if len(vars) == 3 {
				populateStatus(vars[1], vars[2], status)
				continue
			}

			vars = rStatusNodeIp.FindStringSubmatch(line)
			if len(vars) == 2 {
				status.Nodes = append(status.Nodes, vars[1])
			}
		}
		return trace.ConvertSystemError(scanner.Err())
	}
}

func populateStatus(key, value string, status *GravityStatus) error {
	switch key {
	case "Cluster":
		status.Cluster = value
	case "Join token":
		status.Token = value
	case "Application":
		status.Application = value
	case "Status":
		status.Status = value
	default:
	}
	return nil
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
func ParseDDOutput(output string) (uint64, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		return 0, trace.BadParameter("expected 3 lines but got %v:\n%v", len(lines), output)
	}

	// 1073741824 bytes (1.1 GB) copied, 4.52455 s, 237 MB/s
	// 1073741824 bytes (1,1 GB, 1,0 GiB) copied, 4,53701 s, 237 MB/s
	testResults := lines[2]
	match := speedRe.FindStringSubmatch(testResults)
	if len(match) != 2 {
		return 0, trace.BadParameter("failed to match speed value (e.g. 237 MB/s) in %q", testResults)
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
