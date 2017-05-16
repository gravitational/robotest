package sshutils

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	keyFile, err := os.Open("/Users/denismishin/.ssh/id_rsa")
	require.NoError(t, err, "SSH file")
	defer keyFile.Close()

	client, err := Client("52.228.36.57:22", "robotest", keyFile)
	require.NoError(t, err, "ssh client")

	t.Run("environment", func(t *testing.T) {
		t.Parallel()
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

}

func testEnv(t *testing.T, client *ssh.Client) {
	session, err := client.NewSession()
	require.NoError(t, err)

	out, err := RunAndParse(context.Background(),
		t.Logf,
		session,
		"echo $AWS_SECURE_KEY",
		map[string]string{"AWS_SECURE_KEY": "SECUREKEY"},
		func(r *bufio.Reader) (interface{}, error) {
			return r.ReadString('\n')
		})
	assert.NoError(t, err)
	assert.Equal(t, "SECUREKEY\n", out)
}

func testTimeout(t *testing.T, client *ssh.Client) {
	session, err := client.NewSession()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = RunAndParse(ctx,
		t.Logf,
		session,
		"sleep 100",
		nil,
		parseDiscard)
	assert.NoError(t, err)
}

func testExitErr(t *testing.T, client *ssh.Client) {
	session, err := client.NewSession()
	require.NoError(t, err)

	_, err = RunAndParse(context.Background(), t.Logf, session, "ls /nosuchcommand", nil, parseDiscard)
	assert.Error(t, err)
	assert.IsType(t, &trace.TraceErr{}, err)
}

func parseDiscard(r *bufio.Reader) (interface{}, error) {
	io.Copy(ioutil.Discard, r)
	return nil, nil
}
