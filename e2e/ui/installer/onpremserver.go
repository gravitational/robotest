package installer

import (
	"fmt"

	. "github.com/onsi/gomega"

	"github.com/sclevine/agouti"
	am "github.com/sclevine/agouti/matchers"
)

type OnPremServer struct {
	Hostname string
	index    int
	page     *agouti.Page
}

func (p *OnPremServer) GetIPs() []string {
	var result []string
	js := ` var result = []; var cssSelector = "%v .grv-provision-req-server-interface li a"; var children = document.querySelectorAll(cssSelector); children.forEach( z => result.push(z.text) ); return result; `
	js = fmt.Sprintf(js, getServerCssSelector(p.index))
	p.page.RunScript(js, nil, &result)
	return result
}

func createServer(page *agouti.Page, index int) OnPremServer {
	cssSelector := getServerCssSelector(index)
	el := page.Find(cssSelector)
	Expect(el).To(am.BeFound())

	hostName, _ := el.Find(".grv-provision-req-server-hostname span").Text()
	Expect(hostName).NotTo(BeEmpty())

	return OnPremServer{page: page, Hostname: hostName, index: index}

}

func getServerCssSelector(index int) string {
	return fmt.Sprintf(".grv-provision-req-server:nth-child(%v)", index+1)
}
