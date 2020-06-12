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

package debug

import (
	"net/http"
	_ "net/http/pprof"

	log "github.com/sirupsen/logrus"
)

// StartProfiling starts profiling endpoint, will return AlreadyExists
// if profiling has been initiated
func StartProfiling(httpEndpoint string) {
	log.Infof("[PROFILING] http %v", httpEndpoint)

	go func() {
		log.Println(http.ListenAndServe(httpEndpoint, nil))
	}()
}
