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

import (
	"fmt"
	"reflect"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/trace"
)

func TestCreatesPool(t *testing.T) {
	// setup
	nodes := []Node{&node{addr: "a"}, &node{addr: "b"}}
	allocated := []string{"a"}

	// exercise
	pool := NewNodePool(nodes, allocated)

	// verify
	if pool.Size() != len(nodes) {
		t.Errorf("expected %v nodes but got %v", len(nodes), pool.Size())
	}
	if pool.SizeAllocated() != len(allocated) {
		t.Errorf("expected %v nodes but got %v", len(allocated), pool.SizeAllocated())
	}
}

func TestAllocatesAndFrees(t *testing.T) {
	// setup
	nodes := []Node{node{addr: "a"}, node{addr: "b"}, node{addr: "c"}}
	allocated := []string{"a", "b"}

	// exercise
	pool := NewNodePool(nodes, allocated)
	allocatedNodes, err := pool.Allocate(1)

	// verify
	if err != nil {
		t.Errorf("failed to allocate node: %v", err)
	}
	if len(allocatedNodes) == 0 {
		t.Error("failed to allocate node")
	}
	if !reflect.DeepEqual(allocatedNodes[0], nodes[2]) {
		t.Errorf("unexpected allocation: want %v, got %v", nodes[1], allocatedNodes[0])
	}
	if pool.SizeAllocated() != len(allocated)+1 {
		t.Errorf("expected %v total allocated nodes but got %v", len(allocated)+1, pool.SizeAllocated())
	}
	if len(allocatedNodes) != 1 {
		t.Errorf("expected 1 allocated node but got %v", len(allocatedNodes))
	}

	// exercise
	err = pool.Free(allocatedNodes)

	// verify
	if err != nil {
		t.Errorf("failed to free: %v", err)
	}
	if pool.SizeAllocated() != len(allocated) {
		t.Errorf("expected %v total allocated nodes but got %v", len(allocated), pool.SizeAllocated())
	}
}

func TestFailsToAllocBeyondCapacity(t *testing.T) {
	// setup
	nodes := []Node{&node{addr: "a"}, &node{addr: "b"}}
	allocated := []string{"a", "b"}

	// exercise
	pool := NewNodePool(nodes, allocated)
	node, err := pool.Allocate(1)

	// verify
	if err == nil {
		t.Error("expected an error")
	}
	if node != nil {
		t.Errorf("expected a nil node, but got %v", node)
	}
	if pool.Size() != len(nodes) {
		t.Errorf("expected pool of size %v but got %v", len(node), pool.Size())
	}
}

func TestDoesnotFreeNonExisting(t *testing.T) {
	// setup
	nodes := []Node{&node{addr: "a"}, &node{addr: "b"}}
	allocated := []string{"a"}

	// exercise
	pool := NewNodePool(nodes, allocated)
	err := pool.Free([]Node{node{"non-existing"}})

	// verify
	if err == nil {
		t.Error("expected an error")
	}
	if pool.SizeAllocated() != len(allocated) {
		t.Errorf("expected %v allocated nodes but got %v", len(allocated), pool.SizeAllocated())
	}
}

type node struct {
	addr string
}

func (r node) String() string      { return fmt.Sprintf("node(%v)", r.addr) }
func (r node) Addr() string        { return r.addr }
func (r node) PrivateAddr() string { return r.addr }
func (r node) Client() (*ssh.Client, error) {
	return nil, trace.BadParameter("not implemented")
}
func (r node) Connect() (*ssh.Session, error) {
	return nil, trace.BadParameter("not implemented")
}
