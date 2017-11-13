package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gravitational/robotest/infra"
	"github.com/gravitational/robotest/lib/constants"
	"github.com/gravitational/robotest/lib/utils"
	"github.com/gravitational/trace"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	log "github.com/sirupsen/logrus"
)

type AzureAuthParam struct {
	ClientId, ClientSecret, TenantId string
}

type AzureToken struct {
	Type  string `json:"token_type"`
	Token string `json:"access_token"`
}

const (
	azureTokenUrl      = "https://login.microsoftonline.com/%s/oauth2/token"
	azureManagementUrl = "https://management.azure.com/subscriptions/%s/resourcegroups/%s?api-version=2016-09-01"
)

// GetAuthToken retrieves OAuth token for an application
func AzureGetAuthToken(ctx context.Context, param AzureAuthParam) (*AzureToken, error) {
	reqUrl := fmt.Sprintf(azureTokenUrl, param.TenantId)
	resp, err := http.PostForm(reqUrl,
		url.Values{
			"grant_type":    {"client_credentials"},
			"resource":      {"https://management.azure.com/"},
			"client_id":     {param.ClientId},
			"client_secret": {param.ClientSecret}})
	if err != nil {
		return nil, trace.Wrap(err, "[POST %s]=%v", reqUrl, err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, trace.Wrap(err, "[read response from POST %s]=%v", reqUrl, err)
	}

	var token AzureToken
	if err = json.Unmarshal(body, &token); err != nil {
		return nil, trace.Wrap(err, "%v : data=%q", err, body)
	}

	return &token, nil
}

// RemoveResourceGroup submits resource group deletion request to Azure
func AzureRemoveResourceGroup(ctx context.Context, token *AzureToken, subscription, group string) error {
	if subscription == "" || group == "" {
		return trace.BadParameter("subscription=%s, group=%v", subscription, group)
	}

	client := &http.Client{}

	reqUrl := fmt.Sprintf(azureManagementUrl, subscription, group)
	req, err := http.NewRequest("DELETE", reqUrl, nil)
	if err != nil {
		return trace.Wrap(err, `[DELETE %s]=%v`, reqUrl, err)
	}

	req = req.WithContext(ctx)
	req.Header.Add("Authorization", fmt.Sprintf("%s %s", token.Type, token.Token))
	resp, err := client.Do(req)
	if err != nil {
		return trace.Wrap(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return trace.Errorf("%v/%s [DELETE %s]", resp.StatusCode, resp.Status, reqUrl)
	}
	return nil
}

// azurePrepare will generate required config files for terraform
// based on whether snapshot is available for given configuration
func (r *terraform) azurePrepare(ctx context.Context) (err error) {
	f, err := os.OpenFile(filepath.Join(r.stateDir, tfVmFilename),
		os.O_WRONLY, constants.SharedReadWriteMask)
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	defer f.Close()

	if r.Config.FromImage == nil {
		err = tfVmInstallTemplate.Execute(f, struct{}{})
		return trace.Wrap(err)
	} else {
		err = tfVmInstallTemplate.Execute(f, struct{ ResourceGroup string }{r.Config.FromImage.ResourceGroup})
		return trace.Wrap(err)
	}
}

// azureToken returns auth token based on Azure config provided
func azureToken(cfg *infra.AzureConfig) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint,
		cfg.TenantId)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	token, err := adal.NewServicePrincipalToken(*oauthConfig,
		cfg.ClientId, cfg.ClientSecret,
		azure.PublicCloud.ResourceManagerEndpoint)
	return token, trace.Wrap(err)
}

type azureVmCapture struct {
	numNodes  int
	config    infra.AzureConfig
	logger    log.FieldLogger
	vmClient  compute.VirtualMachinesClient
	imgClient compute.ImagesClient
}

// NewAzureVmCapture returns
func NewAzureVmCapture(cfg Config, logger log.FieldLogger) (*azureVmCapture, error) {
	vmClient := compute.NewVirtualMachinesClient(cfg.Azure.SubscriptionId)
	imgClient := compute.NewImagesClient(cfg.Azure.SubscriptionId)

	token, err := azureToken(cfg.Azure)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	authorizer := autorest.NewBearerAuthorizer(token)

	vmClient.Authorizer = authorizer
	imgClient.Authorizer = authorizer

	return &azureVmCapture{
		numNodes:  cfg.NumNodes,
		config:    *cfg.Azure,
		vmClient:  vmClient,
		imgClient: imgClient,
		logger:    logger,
	}, nil
}

// CaptureVM captures current state of VM on azure
// by performing the following steps:
// 1. deallocate VM
// 2. make snapshot
func (r *azureVmCapture) CaptureVM(ctx context.Context) error {
	errCh := make(chan error, r.numNodes)
	for i := 0; i < r.numNodes; i++ {
		go func(name string) {
			err := r.deallocateVM(ctx, name)
			if err != nil {
				r.logger.WithError(err).Errorf("deallocating %q", name)
				errCh <- trace.Wrap(err)
				return
			}
			err = r.generalizeVM(ctx, name)
			if err != nil {
				r.logger.WithError(err).Errorf("generalizing %q", name)
				errCh <- trace.Wrap(err)
				return
			}
			err = r.createVmImage(ctx, name)
			if err != nil {
				r.logger.WithError(err).Errorf("create VM image %q", name)
			}
			errCh <- trace.Wrap(err)
		}(fmt.Sprintf("node-%d", i))
	}

	err := utils.CollectErrors(ctx, errCh)
	return trace.Wrap(err)
}

func (r *azureVmCapture) deallocateVM(ctx context.Context, vmName string) error {
	req, err := r.vmClient.DeallocatePreparer(r.config.ResourceGroup, vmName, nil)
	if err != nil {
		return trace.Wrap(err)
	}
	req = req.WithContext(ctx)

	resp, err := r.vmClient.DeallocateSender(req)
	if err != nil {
		return trace.Wrap(err)
	}

	_, err = r.vmClient.DeallocateResponder(resp)
	return trace.Wrap(err)
}

func (r *azureVmCapture) generalizeVM(ctx context.Context, vmName string) error {
	req, err := r.vmClient.GeneralizePreparer(r.config.ResourceGroup, vmName)
	if err != nil {
		return trace.Wrap(err)
	}
	req = req.WithContext(ctx)

	resp, err := r.vmClient.GeneralizeSender(req)
	if err != nil {
		return trace.Wrap(err)
	}

	_, err = r.vmClient.GeneralizeResponder(resp)
	return trace.Wrap(err)
}

func (r *azureVmCapture) createVmImage(ctx context.Context, name string) error {
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s",
		r.config.SubscriptionId, r.config.ResourceGroup, name)
	param := compute.Image{
		Location: &r.config.Location,
		ImageProperties: &compute.ImageProperties{
			SourceVirtualMachine: &compute.SubResource{
				ID: &id,
			}}}

	req, err := r.imgClient.CreateOrUpdatePreparer(r.config.ResourceGroup,
		name, param, nil)
	if err != nil {
		return trace.Wrap(err)
	}
	req = req.WithContext(ctx)

	r.logger.WithField("request", req).Debug("azure create image")

	resp, err := r.imgClient.CreateOrUpdateSender(req)
	if err != nil {
		return trace.Wrap(err)
	}

	_, err = r.imgClient.CreateOrUpdateResponder(resp)
	return trace.Wrap(err)
}
