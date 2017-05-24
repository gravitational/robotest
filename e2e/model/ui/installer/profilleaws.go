package installer

import (
	"fmt"

	"github.com/gravitational/robotest/e2e/model/ui/utils"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

// AWSProfile is ui model for aws server profile
type AWSProfile struct {
	Label string
	Count string
	index int
	page  *web.Page
}

// SetInstanceType sets instance type on this profile
func (p *AWSProfile) SetInstanceType(instanceType string) {
	Expect(instanceType).NotTo(BeEmpty(), "should have a valid instance type")
	cssSelector := fmt.Sprintf("%v .grv-installer-aws-instance-type", getProfileCSSSelector(p.index))
	utils.SetDropdownValue(p.page, cssSelector, instanceType)
}

func createAWSProfile(page *web.Page, index int) AWSProfile {
	cssSelector := fmt.Sprintf("%v .grv-installer-provision-node-count h2", getProfileCSSSelector(index))

	child := page.Find(cssSelector)
	Expect(child).To(BeFound())

	nodeCount, _ := child.Text()
	Expect(nodeCount).NotTo(BeEmpty())

	cssSelector = fmt.Sprintf("%v .grv-installer-provision-node-desc h3", getProfileCSSSelector(index))
	Expect(child).To(BeFound())
	nodeLabel, _ := child.Text()

	Expect(nodeLabel).NotTo(BeEmpty())

	return AWSProfile{
		page:  page,
		Count: nodeCount,
		Label: nodeLabel,
	}
}
