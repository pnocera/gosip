package api

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/pnocera/gosip"
)

//go:generate ggen -ent Groups -item Group -conf -coll -mods Select,Expand,Filter,Top,OrderBy -helpers Data,Normalized

// Groups represent SharePoint Site Groups API queryable collection struct
// Always use NewGroups constructor instead of &Groups{}
type Groups struct {
	client    *gosip.SPClient
	config    *RequestConfig
	endpoint  string
	modifiers *ODataMods
}

// GroupsResp - groups response type with helper processor methods
type GroupsResp []byte

// NewGroups - Groups struct constructor function
func NewGroups(client *gosip.SPClient, endpoint string, config *RequestConfig) *Groups {
	return &Groups{
		client:    client,
		endpoint:  endpoint,
		config:    config,
		modifiers: NewODataMods(),
	}
}

// ToURL gets endpoint with modificators raw URL
func (groups *Groups) ToURL() string {
	return toURL(groups.endpoint, groups.modifiers)
}

// Get gets site Groups response - a collection of GroupInfo
func (groups *Groups) Get() (GroupsResp, error) {
	client := NewHTTPClient(groups.client)
	return client.Get(groups.ToURL(), groups.config)
}

// Add creates new group with a specified name. Additional metadata can optionally be provided as string map object.
// `metadata` should correspond to SP.Group type.
func (groups *Groups) Add(title string, metadata map[string]interface{}) (GroupResp, error) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["__metadata"] = map[string]string{
		"type": "SP.Group",
	}
	metadata["Title"] = title
	body, _ := json.Marshal(metadata)
	client := NewHTTPClient(groups.client)
	return client.Post(groups.endpoint, bytes.NewBuffer(body), groups.config)
}

// GetByID gets a group object by its ID
func (groups *Groups) GetByID(groupID int) *Group {
	return NewGroup(
		groups.client,
		fmt.Sprintf("%s/GetById(%d)", groups.endpoint, groupID),
		groups.config,
	)
}

// GetByName gets a group object by its name
func (groups *Groups) GetByName(groupName string) *Group {
	return NewGroup(
		groups.client,
		fmt.Sprintf("%s/GetByName('%s')", groups.endpoint, groupName),
		groups.config,
	)
}

// RemoveByID deletes a group object by its ID
func (groups *Groups) RemoveByID(groupID int) error {
	endpoint := fmt.Sprintf("%s/RemoveById(%d)", groups.ToURL(), groupID)
	client := NewHTTPClient(groups.client)
	_, err := client.Post(endpoint, nil, groups.config)
	return err
}

// RemoveByLoginName deletes a group object by its login name
func (groups *Groups) RemoveByLoginName(loginName string) error {
	endpoint := fmt.Sprintf(
		"%s/RemoveByLoginName('%s')",
		groups.endpoint,
		loginName, // url.QueryEscape(loginName),
	)
	client := NewHTTPClient(groups.client)
	_, err := client.Post(endpoint, nil, groups.config)
	return err
}
