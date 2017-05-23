package gravity

import (
	"bufio"
	"io"
	"regexp"

	"github.com/gravitational/trace"
)

// i.e. "Status: active"
var rStatusKV = regexp.MustCompile(`^(?P<key>[\w\s]+)\:\s*(?P<val>[\w\d\_\-]+),*.*`)
var rStatusNodeIp = regexp.MustCompile(`^[\s\w\-\d]+\((?P<ip>[\d\.]+)\).*`)

// parse `gravity status`
func parseStatus(r *bufio.Reader) (interface{}, error) {
	status := &GravityStatus{
		Nodes: []string{},
	}

	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			return status, nil
		}
		if err != nil {
			return nil, trace.Wrap(err)
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

	return status, nil
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
