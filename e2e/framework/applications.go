package framework

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gravitational/robotest/lib/system"
	"github.com/gravitational/trace"

	"github.com/go-yaml/yaml"
	semver "github.com/hashicorp/go-version"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func UpdateApplication() {
	Expect(ConnectToOpsCenter(TestContext.OpsCenterURL, TestContext.ServiceLogin)).To(Succeed())

	nodes := Cluster.Provisioner().NodePool().AllocedNodes()
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

	bumpedVersion := fmt.Sprintf("--version=%v", bump(*version))
	// Import the same package with a new version to emulate update
	cmd = exec.Command("gravity", "--insecure", stateDir, "app", "import", opsURL, bumpedVersion, outputPath)
	Expect(system.Exec(cmd, os.Stderr)).To(Succeed())

	Distribute("gravity update", nodes[0])
}

func ConnectToOpsCenter(opsCenterURL string, login ServiceLogin) error {
	stateDir := fmt.Sprintf("--state-dir=%v", TestContext.StateDir)
	cmd := exec.Command("gravity", "--insecure", stateDir, "ops", "connect", opsCenterURL,
		login.Username, login.Password)
	return trace.Wrap(system.Exec(cmd, io.MultiWriter(os.Stderr, ginkgo.GinkgoWriter)))
}

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
