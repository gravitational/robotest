package infra

import "strings"

// String returns a textual representation of this list of nodes
func (r Nodes) String() string {
	nodes := make([]string, 0, len(r))
	for _, node := range r {
		nodes = append(nodes, node.String())
	}
	return strings.Join(nodes, ",")
}

// Nodes is a list of infrastructure nodes
type Nodes []Node
