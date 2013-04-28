package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/tent/tent-client-go"
)

var meta *tent.MetaPost

func discover() []*request {
	var err error
	meta, err = tent.Discover(os.Args[1])
	if err != nil {
		panic(err)
	}
	return getRequests()
}

func main() {
	examples := make(map[string]*request)
	tent.HTTP.Transport = &roundTripRecorder{roundTripper: tent.HTTP.Transport}

	discoveryReqs := discover()
	examples["discover_head"] = discoveryReqs[0]
	examples["discover_meta"] = discoveryReqs[1]

	res := make(map[string]string)
	for k, v := range examples {
		res[k] = requestMarkdown(v)
	}

	data, _ := json.Marshal(res)
	ioutil.WriteFile(os.Args[2], data, 0644)
}
