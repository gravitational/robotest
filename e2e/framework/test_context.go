package framework

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/gravitational/configure"
	"github.com/gravitational/trace"
)

var TestContext TestContextType

type TestContextType struct {
	ClusterName  string       `json:"cluster_name" env:"ROBO_CLUSTER_NAME"`
	OpsCenterURL string       `json:"ops_url" env:"ROBO_CLUSTER_NAME"`
	StartURL     string       `json:"start_url" env:"ROBO_START_URL"`
	Login        Login        `json:"login"`
	AWS          AWSConfig    `json:"aws"`
	Onprem       OnpremConfig `json:"onprem"`
}

type Login struct {
	Username string `json:"username" env:"ROBO_USERNAME"`
	Password string `json:"password" env:"ROBO_PASSWORD"`
}

type AWSConfig struct {
	AccessKey string `json:"access_key" env:"ROBO_AWS_ACCESS_KEY"`
	SecretKey string `json:"secret_key" env:"ROBO_AWS_SECRET_KEY"`
	Region    string `json:"region" env:"ROBO_AWS_REGION"`
	KeyPair   string `json:"key_pair" env:"ROBO_AWS_KEYPAIR"`
	VPC       string `json:"vpc" env:"ROBO_AWS_VPC"`
}

type OnpremConfig struct {
	Nodes        int    `json:"nodes" env:"ROBO_NODES"`
	InstallNodes int    `json:"install_nodes" env:"ROBO_INSTALL_NODES"`
	InstallerURL string `json:"installer_url" env:"ROBO_INSTALLER_URL"`
	ScriptPath   string `json:"script_path"  env:"ROBO_SCRIPT_PATH"`
}

func init() {
	conf, err := os.Open(os.Getenv("ROBO_CONFIG_FILE"))
	if err != nil {
		panic(fmt.Sprintf("failed to read config file - set path to config as ROBO_CONFIG_FILE"))
	}
	defer conf.Close()
	err = newFileConfig(conf)
	if err != nil {
		panic(fmt.Sprintf("failed to read config file - set path to config as ROBO_CONFIG_FILE"))
	}
}

func newFileConfig(input io.Reader) error {
	d := json.NewDecoder(input)
	err := d.Decode(&TestContext)
	if err != nil {
		return trace.Wrap(err)
	}

	err = configure.ParseEnv(&TestContext)
	if err != nil {
		return trace.Wrap(err)
	}

	// err = config.Validate()
	// if err != nil {
	// 	return nil, trace.Wrap(err)
	// }
	return nil
}
