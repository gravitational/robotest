package infra

import (
	"context"
	"flag"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var gcProjectId = flag.String("gc-project-id", "", "google cloud project ID")

func TestVmRegistry(t *testing.T) {
	if *gcProjectId == "" {
		t.Skip("no google project ID provided")
		return
	}

	ctx := context.Background()

	registry, err := GCSDatastoreVmRegistry(ctx, *gcProjectId, log.StandardLogger())
	require.NoError(t, err, "vm registry")

	checkpoint := "checkpoint"
	var param = struct {
		Param string `json:"p1"`
	}{"value"}
	image := VmImage{"cloud", "region", "group"}

	err = registry.Store(ctx, checkpoint, param, image)
	require.NoError(t, err, "store")

	imgFound, err := registry.Locate(ctx, image.Cloud, checkpoint, param)
	require.NoError(t, err, "locate")
	require.Equal(t, image, *imgFound)
}
