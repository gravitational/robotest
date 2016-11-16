package terraform

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/gravitational/robotest/infra"
	sshutils "github.com/gravitational/robotest/lib/ssh"
	"github.com/gravitational/robotest/lib/system"
	"github.com/gravitational/trace"

	log "github.com/Sirupsen/logrus"
)

func New(stateDir string, config Config) (*terraform, error) {
	err := config.Validate()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &terraform{
		config:   config,
		stateDir: stateDir,
	}, nil
}

func (r *terraform) Create() (*infra.ProvisionerOutput, error) {
	file := filepath.Base(r.config.ScriptPath)
	err := system.CopyFile(r.config.ScriptPath, filepath.Join(r.stateDir, file))
	if err != nil {
		return nil, trace.Wrap(err)
	}

	args := append([]string{"apply"}, getVars(r.config)...)
	cmd := exec.Command("terraform", args...)
	cmd.Dir = r.stateDir
	cmd.Env = os.Environ()

	var out bytes.Buffer
	w := io.MultiWriter(os.Stdout, &out)

	err = system.Exec(cmd, w)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// find installer public IP
	match := reInstallerIP.FindStringSubmatch(out.String())
	if len(match) != 2 {
		return nil, trace.NotFound(
			"failed to extract installer IP from terraform output: %v", match)
	}
	installerIP := strings.TrimSpace(match[1])

	// find all nodes' private IPs
	match = rePrivateIPs.FindStringSubmatch(out.String())
	if len(match) != 2 {
		return nil, trace.NotFound(
			"failed to extract private IPs from terraform output: %v", match)
	}
	privateIPs := strings.Split(strings.TrimSpace(match[1]), " ")

	// find all nodes' public IPs
	match = rePublicIPs.FindStringSubmatch(out.String())
	if len(match) != 2 {
		return nil, trace.NotFound(
			"failed to extract public IPs from terraform output: %v", match)
	}
	publicIPs := strings.Split(strings.TrimSpace(match[1]), " ")

	return &infra.ProvisionerOutput{
		InstallerIP: installerIP,
		PrivateIPs:  privateIPs,
		PublicIPs:   publicIPs,
	}, nil
}

func (r *terraform) Destroy() error {
	log.Infof("destroying infrastructure: %v", r.stateDir)
	args := append([]string{"destroy", "-force"}, getVars(r.config)...)
	cmd := exec.Command("terraform", args...)
	cmd.Dir = r.stateDir
	return trace.Wrap(system.Exec(cmd, os.Stdout))
}

func (r *terraform) SelectInterface(output infra.ProvisionerOutput, addrs []string) (int, error) {
	// TODO: fallback to the first available address
	return 0, nil
}

// Connect establishes an SSH connection to the specified address
func (r *terraform) Connect(addrIP string) (*ssh.Session, error) {
	keyFile, err := os.Open(r.config.SSHKeyPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return sshutils.Connect(fmt.Sprintf("%v:22", addrIP), "centos", keyFile)
}

func (r *terraform) StartInstall(session *ssh.Session) error {
	return session.Start(installerCommand)
}

func (r *terraform) Nodes() []infra.Node {
	// TODO
	return nil
}

func (r *terraform) Allocate() (infra.Node, error) {
	// TODO
	return nil, nil
}

func (r *terraform) Deallocate(infra.Node) error {
	// TODO: put node back to node pool
	return nil
}

// getVars returns a list of variables to provide to terraform apply/destroy commands
// extracted from the config
func getVars(config Config) []string {
	variables := map[string]string{
		"access_key":    config.AccessKey,
		"secret_key":    config.SecretKey,
		"region":        config.Region,
		"key_pair":      config.KeyPair,
		"instance_type": config.InstanceType,
		"cluster_name":  config.ClusterName,
		"installer_url": config.InstallerURL,
	}
	if config.Nodes != 0 {
		variables["nodes"] = strconv.Itoa(config.Nodes)
	}
	var args []string
	for k, v := range variables {
		if strings.TrimSpace(v) != "" {
			args = append(args, "-var", fmt.Sprintf("%v=%v", k, v))
		}
	}
	return args
}

type terraform struct {
	config   Config
	stateDir string
}

var (
	reInstallerIP = regexp.MustCompile("(?m:^installer_ip = ([0-9\\.]+))")
	rePrivateIPs  = regexp.MustCompile("(?m:^private_ips = ([0-9\\. ]+))")
	rePublicIPs   = regexp.MustCompile("(?m:^public_ips = ([0-9\\. ]+))")
)

// installerCommand waits for the installer tarball to download, unpacks it and launches the installation
const installerCommand = `while [ ! -f /home/centos/installer.tar.gz ]; do sleep 5; done; \
tar -xvf /home/centos/installer.tar.gz -C /home/centos/installer; \
/home/centos/installer/install`
