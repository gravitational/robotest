package xlog

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"

	"github.com/gravitational/trace"
)

const (
	urlShortenerScope = "https://www.googleapis.com/auth/urlshortener"
	shortenerEndpoint = "https://www.googleapis.com/urlshortener/v1/url"
)

type shortenerMsg struct {
	Long  string `json:"longUrl"`
	Short string `json:"id,omitempty"`
	Kind  string `json:"kind,omitempty"`
}

// Google URL shortener has been discontinued.
// See https://developers.googleblog.com/2018/03/transitioning-google-url-shortener.html for details.
// TODO(dmitri): remove or update to use another service
func (c GCLClient) Shorten(ctx context.Context, url string) (short string, err error) {
	var buf bytes.Buffer

	msg := shortenerMsg{Long: url}
	err = json.NewEncoder(&buf).Encode(msg)
	if err != nil {
		return "", trace.Wrap(err, "Encoding URL")
	}

	resp, err := c.shortenerClient.Post(shortenerEndpoint, "application/json", &buf)
	if err != nil {
		return "", trace.Wrap(err, "POST %s failed: %v", shortenerEndpoint, err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", trace.Wrap(err, "reading response from URL shortener")
	}

	err = json.Unmarshal(data, &msg)
	if err != nil {
		return "", trace.Wrap(err, "Decoding response %q", data)
	}

	return msg.Short, nil
}
