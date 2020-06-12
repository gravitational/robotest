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

package site

import (
	"encoding/json"

	"github.com/gravitational/robotest/e2e/uimodel/utils"

	. "github.com/onsi/gomega"
	. "github.com/sclevine/agouti/matchers"
)

// AppPage is cluster index page ui model
type AppPage struct {
	site *Site
}

// AppVersion is application version
type AppVersion struct {
	Index        int
	Version      string
	ReleaseNotes string
}

// StartUpdateOperation starts update operation
func (a *AppPage) StartUpdateOperation(toVersion AppVersion) {
	allButtons := a.site.page.All(".grv-site-app-new-ver .btn-primary")
	button := allButtons.At(toVersion.Index)

	Expect(button).To(BeFound())
	Expect(button.Click()).To(Succeed(), "should click on update app button")

	utils.PauseForComponentJs()
	Expect(a.site.page.Find(".grv-site-dlg-update-app .btn-warning").Click()).
		To(Succeed(), "should click on update app confirmation button")

	a.site.WaitForBusyState()
}

// GetCurrentVersion returns current installed version of application
func (a *AppPage) GetCurrentVersion() AppVersion {
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

	Expect(a.site.page.RunScript(script, nil, &jsOutput)).ShouldNot(HaveOccurred(), expectDescriptionText)
	Expect(json.Unmarshal([]byte(jsOutput), &version)).ShouldNot(HaveOccurred(), expectDescriptionText)

	return version
}

// GetNewVersions returns this cluster new application versions
func (a *AppPage) GetNewVersions() []AppVersion {
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

	Expect(a.site.page.RunScript(script, nil, &jsOutput)).ShouldNot(HaveOccurred(), expectDescriptionText)
	Expect(json.Unmarshal([]byte(jsOutput), &versions)).ShouldNot(HaveOccurred(), expectDescriptionText)

	return versions
}
