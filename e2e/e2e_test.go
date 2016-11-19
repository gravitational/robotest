package e2e

import (
	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/specs"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("integration tests", func() {

	/*
		if some_aws_flag {
			specs.VerifyAwsInstall(getPage, framework.TestContext)
			specs.VerifyAwsSite(getPage, framework.TestContext)
		}

		if some_onrem_flag {
			specs.VerifyOnpremInstall(getPage, framework.TestContext, cluster)
		}

	*/

	specs.VerifyAwsInstall(getPage, framework.TestContext)
	specs.VerifyAwsSite(getPage, framework.TestContext)
})
