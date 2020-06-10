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
		return nil, trace.NotFound("cannot allocate %v node(s): capacity exceeded (by %v)",
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
