package debug

import (
	"net/http"
	_ "net/http/pprof"

	log "github.com/Sirupsen/logrus"
)

// StartProfiling starts profiling endpoint, will return AlreadyExists
// if profiling has been initiated
func StartProfiling(httpEndpoint string) {
	log.Infof("[PROFILING] http %v", httpEndpoint)

	go func() {
		log.Println(http.ListenAndServe(httpEndpoint, nil))
	}()
}
