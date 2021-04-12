package vkbot

import (
	"encoding/json"
	"fmt"
	"github.com/karlseguin/typed"
	"io/ioutil"
	"net/http"
)

// VkAPI wraps vk api methods to call
type VkAPI interface {
	// CallMethod calls api.vk.com method by name with params
	CallMethod(methodName string, params Params) (typed.Typed, error)
}

type vkAPI struct {
	Version  string
	Language string
	URL      string
	Token    string

	client *http.Client
}

// NewVkAPI create new vk api with token
// and default version
func NewVkAPI(token string) VkAPI {
	vkAPI := &vkAPI{
		Version: VkAPIVersion,
		URL:     VkAPIUrl,
		Token:   token,
		client:  client,
	}
	return vkAPI
}

func (api *vkAPI) SetLanguage(lang string) {
	api.Language = lang
}

func (api *vkAPI) SetToken(token string) {
	api.Token = token
}

func (api *vkAPI) CallMethod(methodName string, params Params) (typed.Typed, error) {
	params["v"] = api.Version
	params["lang"] = api.Language
	params["access_token"] = api.Token

	values := params.URLValues()
	httpResp, err := api.client.PostForm(api.URL + methodName, values)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	data := typed.Typed{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	if _, ok := data["error"]; ok {
		err := newInternalError(fmt.Errorf("vk api error response"), "method called %s", methodName)
		err.Misc["resp"] = data
		return nil, err
	}
	if _, ok := data["response"]; !ok {
		err := newInternalError(fmt.Errorf("vk api invalid response"), "method called %s", methodName)
		err.Misc["resp"] = data
		return nil, err
	}
	if _, ok := data.IntIf("response"); ok {
		return data, nil
	}
	resp, ok := data.ObjectIf("response")
	if !ok {
		err := newInternalError(fmt.Errorf("response field not a json object"), "method called %s", methodName)
		err.Misc["response"] = data
		return nil, err
	}
	return resp, nil
}
