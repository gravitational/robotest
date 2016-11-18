package common

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sclevine/agouti"
	am "github.com/sclevine/agouti/matchers"
)

func SetDropDownValue(page *agouti.Page, classPath string, value string) {
	if !strings.HasPrefix(classPath, ".") {
		classPath = "." + classPath
	}

	var result []string
	page.Find(classPath).Click()

	js := ` var result = []; var cssSelector = "%v .Select-option"; var children = document.querySelectorAll(cssSelector); children.forEach( z => result.push(z.innerText) ); return result; `
	js = fmt.Sprintf(js, classPath)

	page.RunScript(js, nil, &result)

	for index, optionValue := range result {
		if optionValue == value {
			optionClass := fmt.Sprintf("%v .Select-option:nth-child(%v)", classPath, index+1)
			Expect(page.Find(optionClass)).To(am.BeFound())
			Expect(page.Find(optionClass).Click()).To(Succeed())
			return
		}
	}

	failText := fmt.Sprintf("Unable to make dropdown selection for value - %v", value)
	Fail(failText)
}

func FillOutAwsKeys(page *agouti.Page, accessKey string, secretKey string) {
	Expect(page.FindByName("aws_access_key").Fill(accessKey)).To(
		Succeed(),
		"should enter access key")

	Expect(page.FindByName("aws_secret_key").Fill(secretKey)).To(
		Succeed(),
		"should enter secret key")

}

func Pause(params ...time.Duration) {
	// default value
	timeInterval := 100 * time.Millisecond

	if len(params) != 0 {
		timeInterval = params[0]
	}

	time.Sleep(timeInterval)
}
