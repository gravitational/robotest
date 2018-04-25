package gravity

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDDOutputParser(t *testing.T) {
	flag.Parse()

	var testCases = []struct {
		input       string
		expectedBps uint64
		comment     string
	}{
		{
			input: `1024+0 records in
1024+0 records out
104857600 bytes (105 MB, 100 MiB) copied, 3.06473 s, 237 MB/s`,
			expectedBps: 237 * 1000000,
			comment:     "parses the required value",
		},
		{
			input: `sudo: unable to resolve host node-0
1024+0 records in
1024+0 records out
104857600 bytes (105 MB, 100 MiB) copied, 3.06473 s, 2 GB/s`,
			expectedBps: 2 * 1000000000,
			comment:     "ignores unrelevant parts",
		},
		{
			input: `1024+0 records in
1024+0 records out
104857600 bytes (105 MB, 100 MiB) copied, 3.06473 s, 2048 kB/s`,
			expectedBps: 2048 * 1000,
			comment:     "also handles kilobytes/sec",
		},
	}

	for _, testCase := range testCases {
		bps, err := ParseDDOutput(testCase.input)
		require.Nil(t, err, testCase.comment)
		assert.Equal(t, bps, testCase.expectedBps, testCase.comment)
	}
}
