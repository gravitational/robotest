package sshutils

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sshTestHost = flag.String("host", "", "SSH test host address")
var sshTestUser = flag.String("user", "robotest", "SSH test user")
var sshTestKeyPath = flag.String("key", "", "Path to SSH private key")

func TestSshUtils(t *testing.T) {
	flag.Parse()

	require.NotEmpty(t, *sshTestHost, "ssh host")
	require.NotEmpty(t, *sshTestKeyPath, "ssh key")
	require.NotEmpty(t, *sshTestUser, "ssh user")

	keyFile, err := os.Open(*sshTestKeyPath)
	require.NoError(t, err, "SSH file")
	defer keyFile.Close()

	client, err := Client(fmt.Sprintf("%s:22", *sshTestHost), *sshTestUser, keyFile)
	require.NoError(t, err, "ssh client")

	t.Run("environment", func(t *testing.T) {
		t.Parallel()
		t.Skip() // this requires setup on sshd side, and we no longer use this method
		testEnv(t, client)
	})

	t.Run("timeout", func(t *testing.T) {
		t.Parallel()
		testTimeout(t, client)
	})

	t.Run("exit error", func(t *testing.T) {
		t.Parallel()
		testExitErr(t, client)
	})

	t.Run("test file", func(t *testing.T) {
		t.Parallel()
		testFile(t, client)
	})

	t.Run("scp", func(t *testing.T) {
		t.Parallel()
		testPutFile(t, client)
	})
}

func testPutFile(t *testing.T, client *ssh.Client) {
	p, err := PutFile(context.Background(), t.Logf, client,
		"/bin/echo", "/tmp")
	assert.NoError(t, err)
	assert.EqualValues(t, "/tmp/echo", p, "path")
}

func testEnv(t *testing.T, client *ssh.Client) {
	var out string
	exit, err := RunAndParse(context.Background(),
		t.Logf,
		client,
		"echo $AWS_SECURE_KEY",
		// NOTE: add `AcceptEnv AWS_*` to /etc/ssh/sshd.conf
		map[string]string{"AWS_SECURE_KEY": "SECUREKEY"},
		func(r *bufio.Reader) (err error) {
			out, err = r.ReadString('\n')
			return trace.Wrap(err)
		})
	assert.NoError(t, err)
	assert.Zero(t, exit, "exit code")
	assert.Equal(t, "SECUREKEY\n", out)
}

func testTimeout(t *testing.T, client *ssh.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	_, exit, err := RunAndParse(ctx,
		t.Logf,
		client,
		"sleep 100",
		nil,
		ParseDiscard)
	assert.Error(t, err)
	assert.Equal(t, exitStatusUndefined, exit)
}

func testExitErr(t *testing.T, client *ssh.Client) {
	_, exit, err := RunAndParse(context.Background(), t.Logf, client, "false", nil, ParseDiscard)
	assert.NoError(t, err)
	assert.NotZero(t, exit)
}

func testFile(t *testing.T, client *ssh.Client) {
	ctx := context.Background()

	err := TestFile(ctx, t.Logf, client, "/", TestDir)
	assert.NoError(t, err, TestDir)

	err = TestFile(ctx, t.Logf, client, "/nosuchfile", TestRegularFile)
	assert.True(t, trace.IsNotFound(err))

	err = TestFile(ctx, t.Logf, client, "/", "-nosuchflag")
	assert.True(t, err != nil && !trace.IsNotFound(err), "invalid flag")
}
