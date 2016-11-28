package installer

import (
	"fmt"

	"github.com/gravitational/robotest/e2e/model/ui"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type AWSProfile struct {
	Label string
	Count string
	index int
	page  *web.Page
}

func FindAWSProfiles(page *web.Page) []AWSProfile {
	var profiles []AWSProfile

	reqs := page.All(".grv-installer-provision-reqs-item")
	elements, err := reqs.Elements()
	Expect(err).NotTo(HaveOccurred())

	for i := range elements {
		profiles = append(profiles, createAWSProfile(page, i))
	}

	return profiles
}

func (p *AWSProfile) SetInstanceType(instanceType string) {
	Expect(instanceType).NotTo(BeEmpty(), "should have a valid instance type")
	cssSelector := fmt.Sprintf("%v .grv-installer-aws-instance-type", getProfileCssSelector(p.index))
	ui.SetDropdownValue(p.page, cssSelector, instanceType)
}

func getAWSProfileCssSelector(index int) string {
	return fmt.Sprintf(".grv-installer-provision-reqs-item:nth-child(%v)", index+1)
}

func createAWSProfile(page *web.Page, index int) AWSProfile {
	cssSelector := fmt.Sprintf("%v .grv-installer-provision-node-count h2", getProfileCssSelector(index))

	child := page.Find(cssSelector)
	Expect(child).To(BeFound())

	nodeCount, _ := child.Text()
	Expect(nodeCount).NotTo(BeEmpty())

	cssSelector = fmt.Sprintf("%v .grv-installer-provision-node-desc h3", getProfileCssSelector(index))
	Expect(child).To(BeFound())
	nodeLabel, _ := child.Text()

	Expect(nodeLabel).NotTo(BeEmpty())

	return AWSProfile{
		page:  page,
		Count: nodeCount,
		Label: nodeLabel,
	}
}
