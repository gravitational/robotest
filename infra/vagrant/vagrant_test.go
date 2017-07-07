package vagrant

import (
	"reflect"
	"testing"

	"github.com/gravitational/robotest/infra"
	log "github.com/sirupsen/logrus"
)

func TestParsesSSHConfig(t *testing.T) {
	var testCases = []struct {
		comment  string
		config   []byte
		expected []infra.Node
	}{
		{
			comment: "Parses quoted identity file path",
			config: []byte(`Host master
  HostName 127.0.0.1
  User vagrant
  Port 2222
  UserKnownHostsFile /dev/null
  StrictHostKeyChecking no
  PasswordAuthentication no
  IdentityFile "/path/to/box/virtualbox/private_key"
  IdentitiesOnly yes
  LogLevel FATAL`),
			expected: []infra.Node{&node{identityFile: "/path/to/box/virtualbox/private_key", addrIP: "127.0.0.1"}},
		},
		{
			comment: "Handles unquoted identity file path as well",
			config: []byte(`Host master
  HostName 127.0.0.1
  User vagrant
  Port 2222
  UserKnownHostsFile /dev/null
  StrictHostKeyChecking no
  PasswordAuthentication no
  IdentityFile /path/to/box/virtualbox/private_key
  IdentitiesOnly yes
  LogLevel FATAL`),
			expected: []infra.Node{&node{identityFile: "/path/to/box/virtualbox/private_key", addrIP: "127.0.0.1"}},
		},
	}
	getIP := func(host string) (string, error) { return "127.0.0.1", nil }

	for _, testCase := range testCases {
		obtained, err := parseSSHConfig(testCase.config, getIP)
		if err != nil {
			t.Errorf("failed to parse SSH config: %v", err)
		}
		log.Infof("obtained: %#v", obtained)
		if !reflect.DeepEqual(obtained, testCase.expected) {
			t.Errorf("expected %v but got %v", testCase.expected, obtained)
		}
	}
}
