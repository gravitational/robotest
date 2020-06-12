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

	"github.com/gravitational/robotest/e2e/uimodel/utils"

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
	child = page.Find(cssSelector)
	Expect(child).To(BeFound())
	nodeLabel, _ := child.Text()

	Expect(nodeLabel).NotTo(BeEmpty())

	return AWSProfile{
		page:  page,
		Count: nodeCount,
		Label: nodeLabel,
	}
}
