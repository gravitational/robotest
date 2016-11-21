package installer

import (
	"fmt"

	. "github.com/onsi/gomega"

	"github.com/sclevine/agouti"
	am "github.com/sclevine/agouti/matchers"
)

type OnPremProfile struct {
	Command string
	Label   string
	Count   string
	index   int
	page    *agouti.Page
}

func FindOnPremProfiles(page *agouti.Page) []OnPremProfile {
	var profiles = []OnPremProfile{}

	s := page.All(".grv-installer-provision-reqs-item")

	elements, _ := s.Elements()

	for index, _ := range elements {
		profiles = append(profiles, createProfile(page, index))
	}

	return profiles
}

func (p *OnPremProfile) GetServers() []OnPremServer {
	var servers = []OnPremServer{}

	s := p.page.All(".grv-provision-req-server")

	elements, _ := s.Elements()

	for index, _ := range elements {
		servers = append(servers, createServer(p.page, index))
	}

	return servers
}

func getProfileCssSelector(index int) string {
	return fmt.Sprintf(".grv-installer-provision-reqs-item:nth-child(%v)", index+1)
}

func createProfile(page *agouti.Page, index int) OnPremProfile {
	cssSelector := fmt.Sprintf("%v .grv-installer-server-instruction span", getProfileCssSelector(index))

	element := page.Find(cssSelector)
	Expect(element).To(am.BeFound())

	command, _ := element.Text()
	Expect(command).NotTo(BeEmpty())

	cssSelector = fmt.Sprintf("%v .grv-installer-provision-node-count h2", getProfileCssSelector(index))

	child := page.Find(cssSelector)
	Expect(child).To(am.BeFound())

	nodeCount, _ := child.Text()
	Expect(nodeCount).NotTo(BeEmpty())

	profile := OnPremProfile{Command: command, page: page, Count: nodeCount}
	return profile
}
