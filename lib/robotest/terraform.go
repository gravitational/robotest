package robotest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/trace"
)

type terraformOutput struct {
	installerIP string
	privateIPs  []string
	publicIPs   []string
}

func provisionTerraform(stateDir, scriptPath string, conf config) (*terraformOutput, error) {
	scriptBytes, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = ioutil.WriteFile(filepath.Join(stateDir, "terraform.tf"), scriptBytes, 0644)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	args := append([]string{"apply"}, getVars(conf)...)
	cmd := exec.Command("terraform", args...)
	cmd.Dir = stateDir
	cmd.Env = os.Environ()

	var out bytes.Buffer
	w := io.MultiWriter(os.Stdout, &out)

	err = utils.Exec(cmd, w)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// find installer public IP
	match := reInstallerIP.FindStringSubmatch(out.String())
	if len(match) != 2 {
		return nil, trace.BadParameter(
			"could not extract installer IP from terraform output: %v", match)
	}
	installerIP := strings.TrimSpace(match[1])

	// find all nodes' private IPs
	match = rePrivateIPs.FindStringSubmatch(out.String())
	if len(match) != 2 {
		return nil, trace.BadParameter(
			"could not extract private IPs from terraform output: %v", match)
	}
	privateIPs := strings.Split(strings.TrimSpace(match[1]), " ")

	// find all nodes' public IPs
	match = rePublicIPs.FindStringSubmatch(out.String())
	if len(match) != 2 {
		return nil, trace.BadParameter(
			"could not extract public IPs from terraform output: %v", match)
	}
	publicIPs := strings.Split(strings.TrimSpace(match[1]), " ")

	return &terraformOutput{
		installerIP: installerIP,
		privateIPs:  privateIPs,
		publicIPs:   publicIPs,
	}, nil
}

func destroyTerraform(stateDir string, conf config) error {
	log.Infof("destroying infrastructure: %v", stateDir)
	args := append([]string{"destroy", "-force"}, getVars(conf)...)
	cmd := exec.Command("terraform", args...)
	cmd.Dir = stateDir
	return trace.Wrap(utils.Exec(cmd, os.Stdout))
}

// getVars returns a list of variables to provide to terraform apply/destroy commands
// extracted from the config
func getVars(conf config) []string {
	variables := map[string]string{
		"access_key":    conf.AccessKey,
		"secret_key":    conf.SecretKey,
		"region":        conf.Region,
		"key_pair":      conf.KeyPair,
		"instance_type": conf.InstanceType,
		"cluster_name":  conf.ClusterName,
		"installer_url": conf.InstallerURL,
	}
	if conf.Nodes != 0 {
		variables["nodes"] = strconv.Itoa(conf.Nodes)
	}
	var args []string
	for k, v := range variables {
		if strings.TrimSpace(v) != "" {
			args = append(args, "-var", fmt.Sprintf("%v=%v", k, v))
		}
	}
	return args
}

var (
	reInstallerIP = regexp.MustCompile("(?m:^installer_ip = ([0-9\\.]+))")
	rePrivateIPs  = regexp.MustCompile("(?m:^private_ips = ([0-9\\. ]+))")
	rePublicIPs   = regexp.MustCompile("(?m:^public_ips = ([0-9\\. ]+))")
)
