package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/gravitational/robotest/e2e/framework"
	"github.com/gravitational/robotest/e2e/uimodel/defaults"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

func IsErrorPage(page *web.Page) bool {
	return checkIfExists(page, ".grv-msg-page")
}

func IsInstaller(page *web.Page) bool {
	return checkIfExists(page, ".grv-installer")
}

func HasValidationErrors(page *web.Page) bool {
	return checkIfExists(page, "label.error")
}

func IsLoginPage(page *web.Page) bool {
	count, _ := page.FindByClass("grv-user-login").Count()
	return count > 0
}

func checkIfExists(page *web.Page, className string) bool {
	el := page.Find(className)
	count, _ := el.Count()
	return count > 0
}

func FormatUrl(page *web.Page, prefix string) string {
	url, err := page.URL()
	Expect(err).NotTo(HaveOccurred())
	return framework.URLPathFromString(url, prefix)
}

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

	framework.Failf("failed to select value %q in dropdown %q", value, classPath)
}

// There are 2 different controls that UI uses for dropdown thus each
// requires different handling
func SetDropdownValue2(page *web.Page, rootSelector, buttonSelector, value string) {
	const scriptTemplate = `
            var options = [];
            var cssSelector = "%v .dropdown-menu a";
            var children = document.querySelectorAll(cssSelector);
            children.forEach( option => options.push(option.innerText) );
            return options;
        `

	if buttonSelector == "" {
		buttonSelector = rootSelector
	} else {
		buttonSelector = fmt.Sprintf("%v %v", rootSelector, buttonSelector)
	}
	Expect(page.Find(buttonSelector).Click()).To(Succeed())

	script := fmt.Sprintf(scriptTemplate, rootSelector)
	var options []string

	Expect(page.RunScript(script, nil, &options)).To(Succeed())

	for index, optionValue := range options {
		if optionValue == value {
			optionClass := fmt.Sprintf("%v li:nth-child(%v) a", rootSelector, index+1)
			Expect(page.Find(optionClass)).To(BeFound())
			Expect(page.Find(optionClass).Click()).To(
				Succeed(),
				"should select given dropdown value")
			return
		}
	}

	framework.Failf("failed to select value %q in dropdown %q", value, rootSelector)
}

func SelectRadio(page *web.Page, selector string, matches valueMatcher) {
	const scriptTemplate = `
            var options = [];
            var cssSelector = "%v span";
            var children = document.querySelectorAll(cssSelector);
            children.forEach(option => options.push(option.innerText));
            return options;
        `

	script := fmt.Sprintf(scriptTemplate, selector)
	var options []string

	Expect(page.RunScript(script, nil, &options)).To(Succeed())

	for i, optionValue := range options {
		if matches(optionValue) {
			optionClass := fmt.Sprintf("%v:nth-child(%v) span", selector, i+1)
			Expect(page.Find(optionClass)).To(BeFound())
			Expect(page.Find(optionClass).Click()).To(
				Succeed(),
				"should select given radio control")
			return
		}
	}

	framework.Failf("failed to select control in %q", selector)
}

// valueMatcher defines an interface to select a value
type valueMatcher func(value string) bool

func FillOutAWSKeys(page *web.Page, accessKey string, secretKey string) {
	Expect(page.FindByName("aws_access_key").Fill(accessKey)).To(
		Succeed(),
		"should enter access key")

	Expect(page.FindByName("aws_secret_key").Fill(secretKey)).To(
		Succeed(),
		"should enter secret key")
}

func PauseForPageJs() {
	time.Sleep(1 * time.Second)
}

func PauseForComponentJs() {
	time.Sleep(100 * time.Microsecond)
}

func PauseForServerListRefresh() {
	time.Sleep(defaults.SiteServerListRefreshTimeout)
}

func Pause(params ...time.Duration) {
	timeInterval := 100 * time.Millisecond
	if len(params) != 0 {
		timeInterval = params[0]
	}

	time.Sleep(timeInterval)
}
