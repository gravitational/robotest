package installer

import (
	"fmt"

	"github.com/gravitational/robotest/e2e/ui/common"
	. "github.com/onsi/gomega"

	"github.com/sclevine/agouti"
	am "github.com/sclevine/agouti/matchers"
)

type AwsProfile struct {
	Label string
	Count string
	index int
	page  *agouti.Page
}

func FindAwsProfiles(page *agouti.Page) []AwsProfile {
	var profiles = []AwsProfile{}

	s := page.All(".grv-installer-provision-reqs-item")

	elements, _ := s.Elements()

	for index, _ := range elements {
		profiles = append(profiles, createAwsProfile(page, index))
	}

	return profiles
}

func (self *AwsProfile) SetInstanceType(instanceType string) {
	cssSelector := fmt.Sprintf("%v .grv-installer-aws-instance-type", getProfileCssSelector(self.index))
	common.SetDropDownValue(self.page, cssSelector, instanceType)
}

func getAwsProfileCssSelector(index int) string {
	return fmt.Sprintf(".grv-installer-provision-reqs-item:nth-child(%v)", index+1)
}

func createAwsProfile(page *agouti.Page, index int) AwsProfile {
	cssSelector := fmt.Sprintf("%v .grv-installer-provision-node-count h2", getProfileCssSelector(index))

	child := page.Find(cssSelector)
	Expect(child).To(am.BeFound())

	nodeCount, _ := child.Text()
	Expect(nodeCount).NotTo(BeEmpty())

	cssSelector = fmt.Sprintf("%v .grv-installer-provision-node-desc h3", getProfileCssSelector(index))
	Expect(child).To(am.BeFound())
	nodeLabel, _ := child.Text()

	Expect(nodeLabel).NotTo(BeEmpty())

	return AwsProfile{
		page:  page,
		Count: nodeCount,
		Label: nodeLabel,
	}
}
