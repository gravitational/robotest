package infra

import "github.com/gravitational/trace"

// NewNodePool creates a new instance of NodePool from specified nodes
// and allocation state
func NewNodePool(nodes []Node, alloced []string) *nodePool {
	nodeMap := make(map[string]Node, len(nodes))
	for _, node := range nodes {
		nodeMap[node.Addr()] = node
	}
	p := &nodePool{
		nodes:     nodeMap,
		allocated: make(map[string]struct{}),
	}
	for _, alloc := range alloced {
		p.allocated[alloc] = struct{}{}
	}
	return p
}

// nodePool implements NodePool
type nodePool struct {
	nodes     map[string]Node
	allocated map[string]struct{}
}

func (r *nodePool) Allocate(amount int) (nodes []Node, err error) {
	if amount+r.SizeAllocated() > r.Size() {
		return nil, trace.NotFound("cannot allocate %v node(s): capacity exhausted (by %v)",
			amount, amount+r.SizeAllocated()-r.Size())
	}
	for _, node := range r.nodes {
		if _, exists := r.allocated[node.Addr()]; amount > 0 && !exists {
			r.allocated[node.Addr()] = struct{}{}
			amount = amount - 1
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

func (r *nodePool) Free(nodes []Node) error {
	for _, node := range nodes {
		if _, exists := r.allocated[node.Addr()]; !exists {
			return trace.NotFound("cannot free unallocated node %q", node.Addr())
		}

		delete(r.allocated, node.Addr())
	}
	return nil
}

func (r *nodePool) Nodes() (nodes []Node) {
	nodes = make([]Node, 0, len(r.nodes))
	for addr := range r.nodes {
		node := r.nodes[addr]
		nodes = append(nodes, node)
	}
	return nodes
}

func (r *nodePool) AllocatedNodes() (nodes []Node) {
	nodes = make([]Node, 0, len(r.allocated))
	for addr := range r.allocated {
		node := r.nodes[addr]
		nodes = append(nodes, node)
	}
	return nodes
}

func (r *nodePool) Node(addr string) (Node, error) {
	if node, exists := r.nodes[addr]; exists {
		return node, nil
	}
	return nil, trace.NotFound("node %q not found", addr)
}

func (r *nodePool) Size() int          { return len(r.nodes) }
func (r *nodePool) SizeAllocated() int { return len(r.allocated) }
