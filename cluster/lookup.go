package cluster

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/v3"
)

type Lookup struct {
	httpClient http.Client
	clusterURL string
}

func NewLookup(clusterURL string) *Lookup {
	return &Lookup{
		httpClient: http.Client{
			Timeout: 5 * time.Second,
		},
		clusterURL: clusterURL,
	}
}

func (c *Lookup) Lookup(input *http.Request) (*client.Cluster, error) {
	clusterID := GetClusterID(input)
	if clusterID == "" {
		return nil, nil
	}

	req, err := http.NewRequest("GET", c.clusterURL+"/"+clusterID, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", getAuthorizationHeader(req))

	cookie := getTokenCookie(input)
	if cookie != nil {
		req.AddCookie(cookie)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer close(resp)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, nil
	}

	cluster := &client.Cluster{}
	if err := json.NewDecoder(resp.Body).Decode(cluster); err != nil {
		return nil, errors.Wrap(err, "Parsing clusters response")
	}

	return cluster, nil
}

func GetClusterID(req *http.Request) string {
	clusterID := req.Header.Get("X-API-Cluster-Id")
	if clusterID != "" {
		return clusterID
	}

	parts := strings.Split(req.URL.Path, "/")
	if len(parts) > 3 && strings.HasPrefix(parts[2], "cluster") {
		return parts[3]
	}

	return ""
}

func getAuthorizationHeader(req *http.Request) string {
	return req.Header.Get("Authorization")
}

func getTokenCookie(req *http.Request) *http.Cookie {
	for _, cookie := range req.Cookies() {
		if cookie.Name == "token" {
			return cookie
		}
	}

	return nil
}

func close(resp *http.Response) error {
	io.Copy(ioutil.Discard, resp.Body)
	return resp.Body.Close()
}
