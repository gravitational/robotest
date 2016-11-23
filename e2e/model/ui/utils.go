package ui

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/model/ui/constants"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

func SetDropdownValue(page *web.Page, classPath string, value string) {
	const scriptTemplate = `
		var result = [];
		var cssSelector = "%v .Select-option";
		var children = document.querySelectorAll(cssSelector);
		children.forEach( z => result.push(z.innerText) );
		return result; `

	if !strings.HasPrefix(classPath, ".") {
		classPath = "." + classPath
	}

	var result []string
	page.Find(classPath).Click()

	script := fmt.Sprintf(scriptTemplate, classPath)

	page.RunScript(script, nil, &result)

	for i, optionValue := range result {
		if optionValue == value {
			optionClass := fmt.Sprintf("%v .Select-option:nth-child(%v)", classPath, i+1)
			Expect(page.Find(optionClass)).To(BeFound())
			Expect(page.Find(optionClass).Click()).To(Succeed())
			return
		}
	}

	framework.Failf("failed to select value %q in dropdown", value)
}

// There are 2 different controls that UI uses for dropdown thus each
// requires different handling  
func SetDropDownValue2(page *web.Page, classPath string, value string) {
	if !strings.HasPrefix(classPath, ".") {
		classPath = "." + classPath
	}

	var result []string
	page.Find(classPath).Click()

	js := ` var result = []; var cssSelector = "%v .dropdown-menu a"; var children = document.querySelectorAll(cssSelector); children.forEach( z => result.push(z.innerText) ); return result; `
	js = fmt.Sprintf(js, classPath)

	page.RunScript(js, nil, &result)

	for index, optionValue := range result {
		if optionValue == value {
			optionClass := fmt.Sprintf("%v li:nth-child(%v) a", classPath, index+1)
			Expect(page.Find(optionClass).Click()).To(
				Succeed(),
				"should select given dropdown value")

			return
		}
	}

	Expect(false).To(BeTrue(), "given dropdown value does not exist")
}

func FillOutAwsKeys(page *web.Page, accessKey string, secretKey string) {
	Expect(page.FindByName("aws_access_key").Fill(accessKey)).To(
		Succeed(),
		"should enter access key")

	Expect(page.FindByName("aws_secret_key").Fill(secretKey)).To(
		Succeed(),
		"should enter secret key")
}

func URLPath(urlS string, path string) string {
	newURL, err := url.Parse(urlS)
	Expect(err).NotTo(HaveOccurred())
	newURL.RawQuery = ""
	newURL.Path = path
	return newURL.String()
}

func PauseForPageJs() {
	time.Sleep(1 * time.Second)
}

func PauseForComponentJs() {
	time.Sleep(100 * time.Microsecond)
}

func Pause(params ...time.Duration) {
	timeInterval := constants.PauseTimeout

	if len(params) != 0 {
		timeInterval = params[0]
	}

	time.Sleep(timeInterval)
}
