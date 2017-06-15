package terraform

import (
	"context"
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
)

var azSubscription = flag.String("subscription", "", "")
var azClientId = flag.String("client_id", "", "")
var azClientSecret = flag.String("client_secret", "", "")
var azTenant = flag.String("tenant", "", "")
var azRemoveGroup = flag.String("remove-group", "", "")

func TestAzureGroupRemoval(t *testing.T) {
	require.NotEmpty(t, *azRemoveGroup, "resource group should not be empty")

	param := AzureAuthParam{
		ClientId:     *azClientId,
		ClientSecret: *azClientSecret,
		TenantId:     *azTenant,
	}

	ctx := context.Background()
	token, err := AzureGetAuthToken(ctx, param)
	require.NoError(t, err, "auth token, param=%v", param)

	err = AzureRemoveResourceGroup(ctx, token, *azSubscription, *azRemoveGroup)
	require.NoError(t, err, "remove group")
}
