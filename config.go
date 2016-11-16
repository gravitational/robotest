package main

import (
	"encoding/json"
	"io"

	"github.com/gravitational/robotest/driver/selenium"
	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/infra/terraform"
	"github.com/gravitational/robotest/infra/vagrant"
	// "github.com/gravitational/robotest/driver/cli"

	"github.com/gravitational/configure"
	"github.com/gravitational/trace"
)

func newFileConfig(input io.Reader) (*fileConfig, error) {
	var config fileConfig
	d := json.NewDecoder(input)
	err := d.Decode(&config)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = configure.ParseEnv(&config)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = config.Validate()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &config, nil
}

func (r *fileConfig) Validate() error {
	var errors []error
	err := r.Config.Validate()
	if err != nil {
		errors = append(errors, err)
	}
	if r.Provisioner.Vagrant != nil {
		err := r.Provisioner.Vagrant.Validate()
		if err != nil {
			errors = append(errors, err)
		}
	}
	if r.Provisioner.Terraform != nil {
		err := r.Provisioner.Terraform.Validate()
		if err != nil {
			errors = append(errors, err)
		}
	}
	return trace.NewAggregate(errors...)

}

type fileConfig struct {
	infra.Config

	// LicensePath defines the path to the license file
	LicensePath string            `json:"license_path"`
	Provisioner provisionerConfig `json:"provisioner"`
	Driver      driverConfig      `json:"driver"`
}

type provisionerConfig struct {
	Vagrant   *vagrant.Config   `json:"vagrant"`
	Terraform *terraform.Config `json:"terraform"`
}

type driverConfig struct {
	Web *selenium.Config `json:"web"`
	// Cli *cli.Config
}
