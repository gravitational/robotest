package terraform

import (
	"context"
	"flag"
	"testing"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/trace"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var azSubscription = flag.String("subscription", "", "")
var azClientId = flag.String("client_id", "", "")
var azClientSecret = flag.String("client_secret", "", "")
var azTenant = flag.String("tenant", "", "")
var azResourceGroup = flag.String("resource-group", "", "")
var azLocation = flag.String("location", "", "")

func TestAzureGroupRemoval(t *testing.T) {
	if *azResourceGroup == "" {
		t.Skip("requires resource group")
	}

	param := AzureAuthParam{
		ClientId:     *azClientId,
		ClientSecret: *azClientSecret,
		TenantId:     *azTenant,
	}

	ctx := context.Background()
	token, err := AzureGetAuthToken(ctx, param)
	require.NoError(t, err, "auth token, param=%v", param)

	err = AzureRemoveResourceGroup(ctx, token, *azSubscription, *azResourceGroup)
	require.NoError(t, err, "remove group")
}

func TestAzureCaptureVM(t *testing.T) {
	if *azResourceGroup == "" {
		t.Skip("requires resource group")
	}

	cfg := Config{
		NumNodes: 3,
		Azure: &infra.AzureConfig{
			SubscriptionId: *azSubscription,
			ClientId:       *azClientId,
			ClientSecret:   *azClientSecret,
			TenantId:       *azTenant,
			ResourceGroup:  *azResourceGroup,
			Location:       *azLocation,
		},
	}

	capture, err := NewAzureVmCapture(cfg, log.StandardLogger())
	require.NoError(t, err, trace.DebugReport(err))

	err = capture.CaptureVM(context.Background())
	require.NoError(t, err, trace.DebugReport(err))
}
