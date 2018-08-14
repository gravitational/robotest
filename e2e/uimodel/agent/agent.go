package agent

import (
	"fmt"

	"github.com/gravitational/robotest/e2e/uimodel/utils"
	"github.com/gravitational/robotest/infra"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

// AgentServer is agent server ui model
type AgentServer struct {
	// Hostname is agent server hostname
	Hostname    string
	cssSelector string
	page        *web.Page
}

// CreateAgentServer finds agent servers on the page and creates ui models
func CreateAgentServer(page *web.Page, cssSelector string) AgentServer {
	elem := page.Find(cssSelector)
	Expect(elem).To(BeFound())

	hostname, _ := elem.Find(".grv-provision-req-server-hostname span").Text()
	Expect(hostname).NotTo(BeEmpty(), "should have a hostname")

	return AgentServer{page: page, Hostname: hostname, cssSelector: cssSelector}
}

// SetIP assigns IP to this server
func (a *AgentServer) SetIP(value string) {
	cssSelector := fmt.Sprintf("%v .grv-provision-req-server-interface", a.cssSelector)
	utils.SetDropdownValue2(a.page, cssSelector, "", value)
}

// NeedsDockerDevice checks if this server needs a dedicated device for dock
func (a *AgentServer) NeedsDockerDevice() bool {
	cssSelector := fmt.Sprintf(`%v input[placeholder="loopback"]`, a.cssSelector)
	count, _ := a.page.Find(cssSelector).Count()
	return count == 1
}

// SetDockerDevice assigns docker device to this server
func (a *AgentServer) SetDockerDevice(value string) {
	cssSelector := fmt.Sprintf(`%v input[placeholder="loopback"]`, a.cssSelector)
	Expect(a.page.Find(cssSelector).Fill(value)).To(
		Succeed(),
		"should set a docker device value")
}

// GetIPs returns the list of all server IPs
func (a *AgentServer) GetIPs() []string {
	const scriptTemplate = `
            var result = [];
            var cssSelector = "%v .grv-provision-req-server-interface li a";
            var children = document.querySelectorAll(cssSelector);
            children.forEach( z => result.push(z.text) );
            return result; `
	var result []string

	script := fmt.Sprintf(scriptTemplate, a.cssSelector)
	a.page.RunScript(script, nil, &result)
	return result
}

func (a *AgentServer) SetIPByInfra(provisioner infra.Provisioner) {
	ips := a.GetIPs()
	if len(ips) < 1 {
		return
	}
	var interfaceIP string
	var node infra.Node
	for _, ip := range ips {
		node, _ = provisioner.NodePool().Node(ip)
		if node != nil {
			interfaceIP = ip
			break
		}
		for _, node := range provisioner.NodePool().Nodes() {
			if node.PrivateAddr() == ip {
				interfaceIP = ip
				break
			}
		}
	}

	descriptionText := fmt.Sprintf("cannot find node matching any of %v IPs", a.Hostname)
	Expect(interfaceIP).NotTo(BeEmpty(), descriptionText)
	a.SetIP(interfaceIP)
}
