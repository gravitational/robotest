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

package gravity

import (
	"context"

	sshutil "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"
)

func (g *gravity) streamLogs(ctx context.Context) error {
	return trace.Wrap(sshutil.Run(ctx, g.Client(), g.Logger().WithField("source", "journalctl"),
		"sudo /bin/journalctl --follow --output=cat", nil))
}

func (g *gravity) streamStartupLogs(ctx context.Context) error {
	return trace.Wrap(sshutil.Run(ctx, g.Client(), g.Logger().WithField("source", "journalctl"),
		"sudo /bin/journalctl --identifier=startup-script --lines=all --output=cat --follow", nil))
}
