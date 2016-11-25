package infra

import (
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
)

// NewNodePool creates a new instance of NodePool from specified nodes
// and allocation state
func NewNodePool(nodes []Node, alloced []string) *nodePool {
	nodeMap := make(map[string]Node, len(nodes))
	for _, node := range nodes {
		nodeMap[node.Addr()] = node
	}
	p := &nodePool{
		nodes:   nodeMap,
		alloced: make(map[string]struct{}),
	}
	for _, alloc := range alloced {
		p.alloced[alloc] = struct{}{}
	}
	return p
}

// nodePool implements NodePool
type nodePool struct {
	nodes   map[string]Node
	alloced map[string]struct{}
}

func (r *nodePool) Allocate(amount int) (nodes []Node, err error) {
	toAlloc := amount
	alloced := make(map[string]struct{})
	for _, node := range r.nodes {
		if _, exists := r.alloced[node.Addr()]; exists {
			alloced[node.Addr()] = struct{}{}
		}
		if _, exists := alloced[node.Addr()]; toAlloc > 0 && !exists {
			alloced[node.Addr()] = struct{}{}
			toAlloc = toAlloc - 1
			nodes = append(nodes, node)
		}
	}
	if toAlloc > 0 {
		return nil, trace.NotFound("cannot allocate %v nodes: capacity exhausted (by %v)", amount, toAlloc)
	}
	log.Infof("allocated: %#v", alloced)
	r.alloced = alloced
	return nodes, nil
}

func (r *nodePool) Free(nodes []Node) error {
	for _, node := range nodes {
		if _, exists := r.alloced[node.Addr()]; !exists {
			return trace.NotFound("cannot free unallocated node %q", node.Addr())
		}

		delete(r.alloced, node.Addr())
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

func (r *nodePool) AllocedNodes() (nodes []Node) {
	nodes = make([]Node, 0, len(r.alloced))
	for addr := range r.alloced {
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

func (r *nodePool) Size() int        { return len(r.nodes) }
func (r *nodePool) SizeAlloced() int { return len(r.alloced) }
