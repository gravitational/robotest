package gravity

import (
	"bufio"
	"io"
	"regexp"

	sshutils "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"
)

// i.e. "Status: active"
var rStatusKV = regexp.MustCompile(`^(?P<key>[\w\s]+)\:\s*(?P<val>[\w\d\_\-]+),*.*`)
var rStatusNodeIp = regexp.MustCompile(`^[\s\w\-\d]+\((?P<ip>[\d\.]+)\).*`)

// parse `gravity status`
func parseStatus(status *GravityStatus) sshutils.OutputParseFn {
	return func(r *bufio.Reader) error {
		for {
			line, err := r.ReadString('\n')
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return trace.Wrap(err)
			}

			vars := rStatusKV.FindStringSubmatch(line)
			if len(vars) == 3 {
				populateStatus(vars[1], vars[2], status)
				continue
			}

			vars = rStatusNodeIp.FindStringSubmatch(line)
			if len(vars) == 2 {
				status.Nodes = append(status.Nodes, vars[1])
				continue
			}

		}

		return nil
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
