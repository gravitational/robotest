package common

import (
	"fmt"
	"strings"

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
