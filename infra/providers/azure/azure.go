package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gravitational/trace"
)

type AuthParam struct {
	ClientId, ClientSecret, TenantId string
}

type Token struct {
	Type  string `json:"token_type"`
	Token string `json:"access_token"`
}

const (
	tokenURL      = "https://login.microsoftonline.com/%s/oauth2/token"
	managementURL = "https://management.azure.com/subscriptions/%s/resourcegroups/%s?api-version=2016-09-01"
)

// GetAuthToken retrieves OAuth token for an application
func GetAuthToken(ctx context.Context, param AuthParam) (*Token, error) {
	reqURL := fmt.Sprintf(tokenURL, param.TenantId)
	resp, err := http.PostForm(reqURL,
		url.Values{
			"grant_type":    {"client_credentials"},
			"resource":      {"https://management.azure.com/"},
			"client_id":     {param.ClientId},
			"client_secret": {param.ClientSecret}})
	if err != nil {
		return nil, trace.Wrap(err, "[POST %s]=%v", reqURL, err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, trace.Wrap(err, "[read response from POST %s]=%v", reqURL, err)
	}

	var token Token
	if err = json.Unmarshal(body, &token); err != nil {
		return nil, trace.Wrap(err, "data=%q, err=%v", body, err)
	}

	return &token, nil
}

// RemoveResourceGroup submits resource group deletion request to Azure
func RemoveResourceGroup(ctx context.Context, token Token, subscription, group string) error {
	if subscription == "" || group == "" {
		return trace.BadParameter("subscription=%s, group=%v", subscription, group)
	}

	client := &http.Client{}
	reqURL := fmt.Sprintf(managementURL, subscription, group)
	req, err := http.NewRequest("DELETE", reqURL, nil)
	if err != nil {
		return trace.Wrap(err, `[DELETE %s]=%v`, reqURL, err)
	}

	req = req.WithContext(ctx)
	req.Header.Add("Authorization", fmt.Sprintf("%s %s", token.Type, token.Token))
	resp, err := client.Do(req)
	if err != nil {
		return trace.Wrap(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNotFound {
		return trace.Errorf("%v/%s [DELETE %s]", resp.StatusCode, resp.Status, reqURL)
	}
	return nil
}
