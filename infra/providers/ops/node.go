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

package ops

import (
	"fmt"

	"github.com/gravitational/robotest/infra"
	sshutils "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"
	"golang.org/x/crypto/ssh"
)

type node struct {
	publicIP   string
	privateIP  string
	sshKeyPath string
	sshUser    string
}

func New(publicIP string, privateIP string, sshUser string, sshKeyPath string) infra.Node {
	res := &node{
		publicIP:   publicIP,
		privateIP:  privateIP,
		sshKeyPath: sshKeyPath,
		sshUser:    sshUser,
	}

	return res
}

func (r *node) Addr() string {
	return r.publicIP
}

func (r *node) PrivateAddr() string {
	return r.privateIP
}

func (r *node) Connect() (*ssh.Session, error) {
	client, err := r.Client()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return client.NewSession()
}

func (r *node) Client() (*ssh.Client, error) {
	signer, err := sshutils.MakePrivateKeySignerFromFile(r.sshKeyPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return sshutils.Client(fmt.Sprintf("%v:22", r.publicIP), r.sshUser, signer)
}

func (r node) String() string {
	return fmt.Sprintf("node(addr=%v, private_addr=%v)", r.publicIP, r.privateIP)
}
