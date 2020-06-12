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

package config

import (
	"encoding/json"
	"time"

	"github.com/gravitational/trace"
)

// Timeout provides "human" json serialization/deserialization such as "1m" or "1h" for time.Duration.
//
// For further information, see:
//   https://github.com/golang/go/issues/10275
//   https://stackoverflow.com/questions/48050945/how-to-unmarshal-json-into-durations
type Timeout struct {
	time.Duration
}

func (d Timeout) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Timeout) UnmarshalJSON(buf []byte) error {
	var data string
	if err := json.Unmarshal(buf, &data); err != nil {
		return trace.BadParameter("cannot parse %q as duration: %v", buf, err)
	}
	dur, err := time.ParseDuration(data)
	if err != nil {
		return trace.BadParameter("cannot parse %q as duration: %v", data, err)
	}
	if dur < 0 {
		return trace.BadParameter("timeout must be >= 0")
	}
	d.Duration = dur
	return nil
}
