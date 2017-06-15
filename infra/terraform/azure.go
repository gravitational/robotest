package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gravitational/trace"
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
