package debug

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"time"

	log "github.com/Sirupsen/logrus"
)

func DumpLoop() {
	var interrupts byte
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	var interruptTimeout <-chan time.Time
L:
	for {
		select {
		case <-interrupt:
			interrupts += 1
			if interrupts > 1 {
				break L
			}
			fmt.Println("Dumping goroutine stacks. Press Ctrl-C again to quit.")
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			interruptTimeout = time.After(2 * time.Second)
		case <-interruptTimeout:
			interruptTimeout = nil
			interrupts = 0
		}
	}

	close(interrupt)
	log.Infof("closing dump loop")
}
