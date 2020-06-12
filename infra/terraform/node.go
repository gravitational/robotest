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

package terraform

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

type node struct {
	owner     *terraform
	publicIP  string
	privateIP string
}

func (r *node) Addr() string {
	return r.publicIP
}

func (r *node) PrivateAddr() string {
	return r.privateIP
}

func (r *node) Connect() (*ssh.Session, error) {
	return r.owner.Connect(fmt.Sprintf("%v:22", r.publicIP))
}

func (r *node) Client() (*ssh.Client, error) {
	return r.owner.Client(fmt.Sprintf("%v:22", r.publicIP))
}

func (r node) String() string {
	return fmt.Sprintf("node(addr=%v)", r.publicIP)
}
