package installer

import (
	"fmt"

	"github.com/gravitational/robotest/e2e/model/ui"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type AwsProfile struct {
	Label string
	Count string
	index int
	page  *web.Page
}

func FindAwsProfiles(page *web.Page) []AwsProfile {
	var profiles []AwsProfile

	reqs := page.All(".grv-installer-provision-reqs-item")
	elements, err := reqs.Elements()
	Expect(err).NotTo(HaveOccurred())

	for i := range elements {
		profiles = append(profiles, createAwsProfile(page, i))
	}

	return profiles
}

func (r *AwsProfile) SetInstanceType(instanceType string) {
	cssSelector := fmt.Sprintf("%v .grv-installer-aws-instance-type", getProfileCssSelector(r.index))
	ui.SetDropdownValue(r.page, cssSelector, instanceType)
}

func getAwsProfileCssSelector(index int) string {
	return fmt.Sprintf(".grv-installer-provision-reqs-item:nth-child(%v)", index+1)
}

func createAwsProfile(page *web.Page, index int) AwsProfile {
	cssSelector := fmt.Sprintf("%v .grv-installer-provision-node-count h2", getProfileCssSelector(index))

	child := page.Find(cssSelector)
	Expect(child).To(BeFound())

	nodeCount, _ := child.Text()
	Expect(nodeCount).NotTo(BeEmpty())

	cssSelector = fmt.Sprintf("%v .grv-installer-provision-node-desc h3", getProfileCssSelector(index))
	Expect(child).To(BeFound())
	nodeLabel, _ := child.Text()

	Expect(nodeLabel).NotTo(BeEmpty())

	return AwsProfile{
		page:  page,
		Count: nodeCount,
		Label: nodeLabel,
	}
}
