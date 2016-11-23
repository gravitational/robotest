package site

import (
	"encoding/json"
	"time"

	utils "github.com/gravitational/robotest/e2e/model/ui"
	"github.com/gravitational/robotest/e2e/model/ui/constants"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

type SiteAppPage struct {
	page *agouti.Page
}

type AppVersion struct {
	Index        int
	Version      string
	ReleaseNotes string
}

func (a *SiteAppPage) UpdateApp(version *AppVersion) {
	allBtns := a.page.All(".grv-site-app-new-ver .btn-primary")
	btn := allBtns.At(version.Index)

	Expect(btn).To(BeFound())

	Expect(btn.Click()).To(
		Succeed(),
		"should click on update app button")

	utils.PauseForComponentJs()

	Expect(a.page.Find(".grv-site-dlg-update-app .btn-warning").Click()).To(
		Succeed(),
		"should click on update app confirmation button",
	)

	a.expecAppUpdateProgressIndicator()

	Expect(a.GetCurrentVersion).To(
		BeEquivalentTo(version.Version),
		"current version should match to new one")
}

func (a *SiteAppPage) GetCurrentVersion() AppVersion {
	var version AppVersion
	var jsOutput string

	expectDescriptionText := "should retrieve the current version"

	js := `
        var str = document.querySelector(".grv-site-app-label-version:first-child");
        var text = str.innerText;
        var version = text.split(" ")[1];    
        return JSON.stringify({
            Index: 0, 
            Version: version.trim()
        })
    `

	Expect(a.page.RunScript(js, nil, &jsOutput)).ShouldNot(HaveOccurred(), expectDescriptionText)
	Expect(json.Unmarshal([]byte(jsOutput), &version)).ShouldNot(HaveOccurred(), expectDescriptionText)

	return version
}

func (a *SiteAppPage) GetNewVersions() []AppVersion {
	var versions []AppVersion
	var jsOutput string

	expectDescriptionText := "should retrieve new versions"
	js := `    
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
	Expect(a.page.RunScript(js, nil, &jsOutput)).ShouldNot(HaveOccurred(), expectDescriptionText)
	Expect(json.Unmarshal([]byte(jsOutput), &versions)).ShouldNot(HaveOccurred(), expectDescriptionText)

	return versions
}

func (a *SiteAppPage) expecAppUpdateProgressIndicator() {
	page := a.page
	Eventually(page.FindByClass("grv-site-app-progres-indicator"), constants.ElementTimeout).Should(
		BeFound(),
		"should find progress indicator")

	Eventually(page.FindByClass("grv-site-app-progres-indicator"), constants.OperationTimeout).ShouldNot(
		BeFound(),
		"should wait for progress indicator to disappear")

	// let JS code to update UI (due to different pull timeouts in the JS code)
	utils.Pause(10 * time.Second)
}
