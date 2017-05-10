package agent

import (
	"fmt"

	utils "github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/infra"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type AgentServer struct {
	Hostname    string
	cssSelector string
	page        *web.Page
}

func CreateAgentServer(page *web.Page, cssSelector string) AgentServer {
	elem := page.Find(cssSelector)
	Expect(elem).To(BeFound())

	hostname, _ := elem.Find(".grv-provision-req-server-hostname span").Text()
	Expect(hostname).NotTo(BeEmpty(), "should have a hostname")

	return AgentServer{page: page, Hostname: hostname, cssSelector: cssSelector}
}

func (a *AgentServer) SetIP(value string) {
	cssSelector := fmt.Sprintf("%v .grv-provision-req-server-interface", a.cssSelector)
	utils.SetDropdownValue2(a.page, cssSelector, "", value)
}

func (a *AgentServer) NeedsDockerDevice() bool {
	cssSelector := fmt.Sprintf(`%v input[placeholder="loopback"]`, a.cssSelector)
	element := a.page.Find(cssSelector)
	count, _ := element.Count()
	return count == 1
}

func (a *AgentServer) SetDockerDevice(value string) {
	cssSelector := fmt.Sprintf(`%v input[placeholder="loopback"]`, a.cssSelector)
	Expect(a.page.Find(cssSelector).Fill(value)).To(
		Succeed(),
		"should set a docker device value")
}

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
	if len(ips) < 2 {
		return
	}
	var node infra.Node
	for _, ip := range ips {
		node, _ = provisioner.NodePool().Node(ip)
		if node != nil {
			break
		}
	}

	descriptionText := fmt.Sprintf("cannot find node matching any of %v IPs", a.Hostname)
	Expect(node).NotTo(BeNil(), descriptionText)
	a.SetIP(node.Addr())
}
