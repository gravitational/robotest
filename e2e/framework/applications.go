package framework

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/system"

	"github.com/go-yaml/yaml"
	"github.com/gravitational/trace"
	semver "github.com/hashicorp/go-version"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// FakeUpdateApplication implements site update test by downloading the application tarball,
// incrementing the version and importing the same tarball with a new version.
//
// It downloads the update from one of the remote nodes before returning to ensure
// that the application update is available
func FakeUpdateApplication() {
	Expect(ConnectToOpsCenter(TestContext.OpsCenterURL, TestContext.ServiceLogin)).To(Succeed())
	Expect(TestContext.Application.Locator).NotTo(BeNil(), "expected a valid application package")

	nodes := Cluster.Provisioner().NodePool().AllocatedNodes()
	if len(nodes) == 0 {
		Failf("expected active nodes in cluster, got none")
	}

	stateDir := fmt.Sprintf("--state-dir=%v", TestContext.StateDir)
	opsURL := fmt.Sprintf("--ops-url=%v", TestContext.OpsCenterURL)
	outputPath := filepath.Join(TestContext.StateDir, "app.tar.gz")
	cmd := exec.Command("gravity", "--insecure", stateDir, "package", "export",
		opsURL, TestContext.Application.String(), outputPath)
	Expect(system.Exec(cmd, os.Stderr)).To(Succeed())

	versionS := TestContext.Application.Version
	if versionS == latestMetaversion {
		var err error
		versionS, err = getResourceVersion(outputPath)
		Expect(err).NotTo(HaveOccurred(), "expected to query application package version from tarball")
	}

	version, err := semver.NewVersion(versionS)
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("expected a version in semver format, got %q", TestContext.Application.Version))

	bumpedVersion := bump(*version)
	bumpedVersionParam := fmt.Sprintf("--version=%v", bumpedVersion)
	// Import the same package with a new version to emulate update
	cmd = exec.Command("gravity", "--insecure", stateDir, "app", "import", opsURL, bumpedVersionParam, outputPath)
	Expect(system.Exec(cmd, os.Stderr)).To(Succeed())
	testState.Application.Version = bumpedVersion
}

// UpdateApplicationWithInstaller impements site update via installer tarball
func UpdateApplicationWithInstaller() {
	Expect(ConnectToOpsCenter(TestContext.OpsCenterURL, TestContext.ServiceLogin)).To(Succeed())
	Expect(testState.ProvisionerState.InstallerAddr).NotTo(BeNil(), "expected a valid installer address")

	provisioner := Cluster.Provisioner()
	installerNode, err := provisioner.NodePool().Node(testState.ProvisionerState.InstallerAddr)
	Expect(err).NotTo(HaveOccurred(), "expected to get installer node from previous provisioner state")

	err = infra.UploadUpdate(context.TODO(), provisioner, installerNode)
	Expect(err).NotTo(HaveOccurred(), "expected upload update operation to be completed")
}

// BackupApplication implements test for backup hook
func BackupApplication() {
	Expect(ConnectToOpsCenter(TestContext.OpsCenterURL, TestContext.ServiceLogin)).To(Succeed())
	Expect(TestContext.Application.Locator).NotTo(BeNil(), "expected a valid application package")
	Expect(TestContext.Extensions.BackupConfig.Addr).NotTo(BeNil(), "expect valid node address for backup operation")
	Expect(TestContext.Extensions.BackupConfig.Path).NotTo(BeNil(), "expect valid path to backup file")

	backupNode, err := Cluster.Provisioner().NodePool().Node(TestContext.Extensions.BackupConfig.Addr)
	if err != nil {
		trace.NotFound("node with address %v not found in config state", TestContext.Extensions.BackupConfig.Addr)
	}
	Distribute(fmt.Sprintf("sudo gravity planet enter -- --notty /usr/bin/gravity -- system backup %s %s", TestContext.Application.String(), TestContext.Extensions.BackupConfig.Path), backupNode)
	UpdateBackupState()
}

// RestoreApplication implements test for restore hook
func RestoreApplication() {
	Expect(ConnectToOpsCenter(TestContext.OpsCenterURL, TestContext.ServiceLogin)).To(Succeed())
	Expect(TestContext.Application.Locator).NotTo(BeNil(), "expected a valid application package")
	Expect(testState.BackupState.Addr).NotTo(BeNil(), "expect valid node address for restore operation")
	Expect(testState.BackupState.Path).NotTo(BeNil(), "expect valid path to backup file")

	backupNode, err := Cluster.Provisioner().NodePool().Node(testState.BackupState.Addr)
	if err != nil {
		trace.NotFound("node with address %v not found in config state", testState.BackupState.Addr)
	}
	Distribute(fmt.Sprintf("sudo gravity planet enter -- --notty /usr/bin/gravity -- system restore %s %s", TestContext.Application.String(), testState.BackupState.Path), backupNode)
}

// ConnectToOpsCenter connects to the Ops Center specified with opsCenterURL using
// specified login
func ConnectToOpsCenter(opsCenterURL string, login ServiceLogin) error {
	stateDir := fmt.Sprintf("--state-dir=%v", TestContext.StateDir)
	cmd := exec.Command("gravity", "--insecure", stateDir, "ops", "connect", opsCenterURL,
		login.Username, login.Password)
	return trace.Wrap(system.Exec(cmd, io.MultiWriter(os.Stderr, ginkgo.GinkgoWriter)))
}

// getResourceVersion retrieves the version of the test application
// from the specified tarball
func getResourceVersion(tarball string) (string, error) {
	f, err := os.Open(tarball)
	if err != nil {
		return "", trace.ConvertSystemError(err)
	}
	defer f.Close()
	rz, err := gzip.NewReader(f)
	if err != nil {
		return "", trace.ConvertSystemError(err)
	}
	defer rz.Close()
	r := tar.NewReader(rz)
	var resourceVersion string
	for {
		var hdr *tar.Header
		hdr, err = r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", trace.ConvertSystemError(err)
		}
		if strings.HasSuffix(hdr.Name, "app.yaml") {
			var manifestBytes []byte
			manifestBytes, err = ioutil.ReadAll(r)
			if err != nil {
				return "", trace.ConvertSystemError(err)
			}
			var manifest manifest
			err = yaml.Unmarshal(manifestBytes, &manifest)
			if err != nil {
				return "", trace.Wrap(err)
			}
			resourceVersion = manifest.Metadata.ResourceVersion
			break
		}
	}
	return resourceVersion, trace.Wrap(err)
}

// bump increments the version specified in v by adding 1 to
// the last segment's value
func bump(v semver.Version) string {
	segments := v.Segments()
	if len(segments) == 0 {
		return v.String()
	}

	var buf bytes.Buffer
	segments = make([]int, len(segments))
	copy(segments, v.Segments())
	segments[len(segments)-1] = segments[len(segments)-1] + 1
	formatted := make([]string, len(segments))
	for i, s := range segments {
		formatted[i] = strconv.Itoa(s)
	}
	fmt.Fprintf(&buf, strings.Join(formatted, "."))
	if v.Prerelease() != "" {
		fmt.Fprintf(&buf, "-%s", v.Prerelease())
	}
	if v.Metadata() != "" {
		fmt.Fprintf(&buf, "+%s", v.Metadata())
	}

	return buf.String()
}

type manifest struct {
	Metadata struct {
		ResourceVersion string `yaml:"resourceVersion"`
	} `yaml:"metadata"`
}

const latestMetaversion = "0.0.0+latest"
