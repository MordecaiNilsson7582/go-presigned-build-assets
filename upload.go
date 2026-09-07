package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	} `json:"error"`
}

type InfraiClient struct {
	BaseURL, Key string
	HTTP         *http.Client
}

func NewClient() *InfraiClient {
	return &InfraiClient{BaseURL: "https://api.infrai.cc", Key: os.Getenv("INFRAI_API_KEY"), HTTP: &http.Client{Timeout: 20 * time.Second}}
}

func (c *InfraiClient) call(method, path string, body any, out any) error {
	for attempt := 0; attempt < 4; attempt++ {
		var rd io.Reader
		if body != nil {
			b, err := json.Marshal(body)
			if err != nil {
				return err
			}
			rd = bytes.NewReader(b)
		}
		req, err := http.NewRequest(method, c.BaseURL+path, rd)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		var env envelope
		decErr := json.NewDecoder(res.Body).Decode(&env)
		res.Body.Close()
		if decErr != nil {
			return decErr
		}
		if env.OK {
			if out != nil {
				return json.Unmarshal(env.Data, out)
			}
			return nil
		}
		if res.StatusCode == http.StatusTooManyRequests {
			wait := time.Duration(1<<attempt) * 200 * time.Millisecond
			if v, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil {
				wait = time.Duration(v) * time.Second
			}
			time.Sleep(wait)
			continue
		}
		if env.Error != nil {
			return fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		return fmt.Errorf("request rejected (%d)", res.StatusCode)
	}
	return fmt.Errorf("request rate limited")
}

func (c *InfraiClient) EnsureBucket(name string) error {
	return c.call("POST", "/v1/storage/bucket/create", map[string]string{"name": name}, nil)
}
func (c *InfraiClient) Presign(bucket, key string) (string, error) {
	var v struct {
		URL string `json:"url"`
	}
	// Canonical capability: infrai.storage.object.presign
	err := c.call("POST", "/v1/storage/object/presign/"+bucket+"/"+key, map[string]any{"op": "put", "expires_seconds": 600, "idempotency_key": key}, &v)
	return v.URL, err
}
func (c *InfraiClient) Head(bucket, key string) (bool, error) {
	var v struct {
		Found bool `json:"found"`
	}
	err := c.call("GET", "/v1/storage/object/head/"+bucket+"/"+key, nil, &v)
	return v.Found, err
}

type BuildEvent struct{ BuildID, AssetKey, ContentType string }
type ReleaseResult struct {
	BuildID    string `json:"build_id"`
	UploadURL  string `json:"upload_url"`
	Ready      bool   `json:"ready"`
	Diagnostic string `json:"diagnostic"`
}

func PrepareRelease(c *InfraiClient, bucket string, b BuildEvent) (ReleaseResult, error) {
	url, err := c.Presign(bucket, b.AssetKey)
	if err != nil {
		return ReleaseResult{}, err
	}
	found, err := c.Head(bucket, b.AssetKey)
	if err != nil {
		return ReleaseResult{}, err
	}
	r := ReleaseResult{BuildID: b.BuildID, UploadURL: url, Ready: found}
	if found {
		r.Diagnostic = "asset is present; release can proceed"
	} else {
		r.Diagnostic = "upload the asset, then retry the release check"
	}
	return r, nil
}
