package site

import (
	"encoding/json"
	"time"

	"github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/constants"

	. "github.com/onsi/gomega"
	web "github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type SiteAppPage struct {
	page *web.Page
}

type AppVersion struct {
	Index        int
	Version      string
	ReleaseNotes string
}

func (a *SiteAppPage) UpdateApp(toVersion AppVersion) {
	allButtons := a.page.All(".grv-site-app-new-ver .btn-primary")
	button := allButtons.At(toVersion.Index)

	Expect(button).To(BeFound())

	Expect(button.Click()).To(
		Succeed(),
		"should click on update app button")

	ui.PauseForComponentJs()

	Expect(a.page.Find(".grv-site-dlg-update-app .btn-warning").Click()).To(
		Succeed(),
		"should click on update app confirmation button",
	)

	a.expectAppUpdateProgressIndicator()

	Expect(a.page.Refresh()).To(Succeed())
	Eventually(a.GetCurrentVersion, constants.FindTimeout).Should(
		BeEquivalentTo(toVersion),
		"current version should match to new one")
}

func (a *SiteAppPage) GetCurrentVersion() AppVersion {
	const expectDescriptionText = "should retrieve the current version"
	const script = `
            var str = document.querySelector(".grv-site-app-label-version:first-child");
            var text = "";
            if (str) {
                text = str.innerText;
            }
            var parts = text.split(" ");
            var version = "";
            if (parts.length > 1) {
                version = parts[1];
            }
            return JSON.stringify({
                Index: 0,
                Version: version.trim()
            })
        `
	var version AppVersion
	var jsOutput string

	Expect(a.page.RunScript(script, nil, &jsOutput)).ShouldNot(HaveOccurred(), expectDescriptionText)
	Expect(json.Unmarshal([]byte(jsOutput), &version)).ShouldNot(HaveOccurred(), expectDescriptionText)

	return version
}

func (a *SiteAppPage) GetNewVersions() []AppVersion {
	const expectDescriptionText = "should retrieve new versions"
	const script = `    
            var data = [];        
            var items = document.querySelectorAll(".grv-site-app-new-ver .grv-site-app-label-version");
            items.forEach( (i, index) => {
                var text = i.innerText;
                var ver = text.split(" ")[1];
                data.push({
                    Index: index, 
                    Version: ver.trim()
                } )
            })
                
            return JSON.stringify(data);
        `
	var versions []AppVersion
	var jsOutput string

	Expect(a.page.RunScript(script, nil, &jsOutput)).ShouldNot(HaveOccurred(), expectDescriptionText)
	Expect(json.Unmarshal([]byte(jsOutput), &versions)).ShouldNot(HaveOccurred(), expectDescriptionText)

	return versions
}

func (a *SiteAppPage) expectAppUpdateProgressIndicator() {
	page := a.page
	Eventually(page.FindByClass("grv-site-app-progres-indicator"), constants.ElementTimeout).Should(
		BeFound(),
		"should find progress indicator")

	Eventually(page.FindByClass("grv-site-app-progres-indicator"), constants.OperationTimeout).ShouldNot(
		BeFound(),
		"should wait for progress indicator to disappear")

	// let JS code update UI (due to different pull timeouts in the JS code)
	ui.Pause(10 * time.Second)
}
