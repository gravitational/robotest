/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package installer

import (
	"fmt"
	"strconv"

	"github.com/gravitational/robotest/e2e/uimodel/agent"

	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	am "github.com/sclevine/agouti/matchers"
)

// OnPremProfile is ui model for onprem server profile
type OnPremProfile struct {
	Command string
	Label   string
	Count   int
	index   int
	page    *agouti.Page
}

// GetAgentServers returns agent servers
func (p *OnPremProfile) GetAgentServers() []agent.AgentServer {
	var agentServers = []agent.AgentServer{}
	cssSelector := fmt.Sprintf("%v .grv-provision-req-server", getProfileCSSSelector(p.index))
	elements, _ := p.page.All(cssSelector).Elements()
	for index := range elements {
		cssAgentServerSelector := fmt.Sprintf("%v:nth-child(%v)", cssSelector, index+1)
		agentServers = append(agentServers, agent.CreateAgentServer(p.page, cssAgentServerSelector))
	}

	return agentServers
}

func getProfileCSSSelector(index int) string {
	return fmt.Sprintf(".grv-installer-provision-reqs-item:nth-child(%v)", index+1)
}

func createProfile(page *agouti.Page, index int) OnPremProfile {
	cssSelector := fmt.Sprintf("%v .grv-installer-server-instruction span", getProfileCSSSelector(index))

	element := page.Find(cssSelector)
	Expect(element).To(am.BeFound())

	command, _ := element.Text()
	Expect(command).NotTo(BeEmpty())

	cssSelector = fmt.Sprintf("%v .grv-installer-provision-node-count h2", getProfileCSSSelector(index))

	child := page.Find(cssSelector)
	Expect(child).To(am.BeFound())

	nodeCountText, _ := child.Text()
	Expect(nodeCountText).NotTo(BeEmpty())

	nodeCount, err := strconv.Atoi(nodeCountText)
	Expect(err).NotTo(HaveOccurred(), "unable to convert node count text field to number")

	profile := OnPremProfile{Command: command, page: page, index: index, Count: nodeCount}
	return profile
}
