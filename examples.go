package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/tent/tent-client-go"
)

var meta *tent.MetaPost
var client *tent.Client

func discover() []*request {
	var err error
	meta, err = tent.Discover(os.Args[1])
	maybePanic(err)
	client = &tent.Client{Servers: meta.Servers}
	return getRequests()
}

func createApp() []*request {
	post := tent.NewAppPost(&tent.App{
		Name: "Example App",
		URL:  "https://app.example.com",
		PostTypes: tent.AppPostTypes{
			Write: []string{"https://tent.io/types/status/v0", "https://tent.io/types/photo/v0"},
			Read:  []string{"https://tent.io/types/app/v0"},
		},
		RedirectURI: "https://app.example.com/oauth",
	})
	err := client.CreatePost(post)
	maybePanic(err)
	client.Credentials, _, err = post.LinkedCredentials()
	maybePanic(err)
	oauthURL, _ := meta.Servers[0].URLs.OAuthURL(post.ID, "d173d2bb868a")
	req, _ := http.NewRequest("GET", oauthURL, nil)
	res, err := tent.HTTP.Transport.RoundTrip(req)
	maybePanic(err)
	u, err := url.Parse(res.Header.Get("Location"))
	maybePanic(err)
	client.Credentials, err = client.RequestAccessToken(u.Query().Get("code"))
	maybePanic(err)
	return getRequests()
}

func newPost() *request {
	post := &tent.Post{
		Type:    "https://tent.io/types/status/v0#",
		Content: []byte(`{"text": "example post"}`),
	}
	err := client.CreatePost(post)
	maybePanic(err)
	return getRequests()[0]
}

func newMultipartPost() []*request {
	post := &tent.Post{
		Type:    "https://tent.io/types/photo/v0#",
		Content: []byte(`{"caption": "example photo"}`),
		Attachments: []*tent.PostAttachment{{
			Name:        "example.jpeg",
			Category:    "photo",
			ContentType: "image/jpeg",
			Data:        strings.NewReader("example attachment data"),
		}},
	}
	err := client.CreatePost(post)
	maybePanic(err)

	_, err = io.Copy(ioutil.Discard, post.Attachments[0])
	maybePanic(err)
	post.Attachments[0].Close()

	return getRequests()
}

func main() {
	examples := make(map[string]*request)
	tent.HTTP.Transport = &roundTripRecorder{roundTripper: tent.HTTP.Transport}

	discoveryReqs := discover()
	examples["discover_head"] = discoveryReqs[0]
	examples["discover_meta"] = discoveryReqs[1]

	appReqs := createApp()
	examples["app_create"] = appReqs[0]
	examples["app_credentials"] = appReqs[1]
	examples["oauth_redirect"] = appReqs[2]
	examples["oauth_token"] = appReqs[3]

	examples["new_post"] = newPost()

	multipartReqs := newMultipartPost()
	examples["new_multipart_post"] = multipartReqs[0]
	examples["get_attachment"] = multipartReqs[1]

	res := make(map[string]string)
	for k, v := range examples {
		res[k] = requestMarkdown(v)
	}

	data, _ := json.Marshal(res)
	ioutil.WriteFile(os.Args[2], data, 0644)
}

func maybePanic(err error) {
	if err != nil {
		if resErr, ok := err.(*tent.BadResponseError); ok && resErr.TentError != nil {
			fmt.Println(resErr.TentError)
		}
		panic(err)
	}
}
