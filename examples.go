package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

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
			Write: []string{"https://tent.io/types/post/v0"},
			Read:  []string{"https://tent.io/types/app/v0"},
		},
		RedirectURI: "https://app.example.com/oauth",
	})
	err := client.CreatePost(post)
	maybePanic(err)
	_, err = post.GetCredentials()
	maybePanic(err)
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
