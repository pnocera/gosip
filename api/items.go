package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pnocera/gosip"
)

//go:generate ggen -ent Items -item Item -conf -coll -mods Select,Expand,Filter,Top,Skip,OrderBy -helpers Data,Normalized,ToMap

// Items represent SharePoint Lists & Document Libraries Items API queryable collection struct
// Always use NewItems constructor instead of &Items{}
type Items struct {
	client    *gosip.SPClient
	config    *RequestConfig
	endpoint  string
	modifiers *ODataMods
}

// ItemsResp - items response type with helper processor methods
type ItemsResp []byte

// NewItems - Items struct constructor function
func NewItems(client *gosip.SPClient, endpoint string, config *RequestConfig) *Items {
	return &Items{
		client:    client,
		endpoint:  endpoint,
		config:    config,
		modifiers: NewODataMods(),
	}
}

// ToURL gets endpoint with modificators raw URL
func (items *Items) ToURL() string {
	return toURL(items.endpoint, items.modifiers)
}

// Get gets Items API queryable collection
func (items *Items) Get() (ItemsResp, error) {
	client := NewHTTPClient(items.client)
	data, err := client.Get(items.ToURL(), items.config)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GetAll gets all items in a list using internal page helper. The use case of the method is getting all the content from large lists.
// Method ignores custom sorting and filtering as not supported for the large lists due to throttling limitations.
func (items *Items) GetAll() ([]ItemResp, error) {
	return getAll(nil, nil, items)
}

// Add adds new item in this list. `body` parameter is byte array representation of JSON string payload relevant to item metadata object.
func (items *Items) Add(body []byte) (ItemResp, error) {
	body = patchMetadataTypeCB(body, func() string {
		endpoint := getPriorEndpoint(items.endpoint, "/Items")
		cacheKey := strings.ToLower(endpoint + "@entitytype")
		if oDataType, found := storage.Get(cacheKey); found {
			return oDataType.(string)
		}
		list := NewList(items.client, endpoint, nil)
		oDataType, _ := list.GetEntityType()
		storage.Set(cacheKey, oDataType, 0)
		return oDataType
	})
	client := NewHTTPClient(items.client)
	return client.Post(items.endpoint, bytes.NewBuffer(body), items.config)
}

// GetByID gets item data object by its ID
func (items *Items) GetByID(itemID int) *Item {
	return NewItem(
		items.client,
		fmt.Sprintf("%s(%d)", items.endpoint, itemID),
		items.config,
	)
}

// GetByCAML gets items data using CAML query
func (items *Items) GetByCAML(caml string) (ItemsResp, error) {
	endpoint := fmt.Sprintf("%s/GetItems", strings.TrimRight(items.endpoint, "/Items"))
	apiURL, _ := url.Parse(endpoint)
	query := url.Values{}
	for k, v := range items.modifiers.Get() {
		query.Add(k, TrimMultiline(v))
	}
	apiURL.RawQuery = query.Encode()

	request := &struct {
		Query struct {
			Metadata struct {
				Type string `json:"type"`
			} `json:"__metadata"`
			ViewXML string `json:"ViewXml"`
		} `json:"query"`
	}{}

	request.Query.Metadata.Type = "SP.CamlQuery"
	request.Query.ViewXML = TrimMultiline(caml)

	body, _ := json.Marshal(request)

	client := NewHTTPClient(items.client)
	return client.Post(apiURL.String(), bytes.NewBuffer(body), items.config)
}

// ToDo:
// Batch

// Helper methods

func getAll(res []ItemResp, cur ItemsResp, items *Items) ([]ItemResp, error) {
	if res == nil && cur == nil {
		itemsCopy := NewItems(items.client, items.endpoint, items.config)
		for key, val := range items.modifiers.Get() {
			switch key {
			case "$select":
				itemsCopy.modifiers.AddSelect(val)
			case "$expand":
				itemsCopy.modifiers.AddExpand(val)
			case "$top":
				top, _ := strconv.Atoi(val)
				itemsCopy.modifiers.AddTop(top)
			}
		}
		itemsResp, err := itemsCopy.Get()
		if err != nil {
			return nil, err
		}
		res = itemsResp.Data()
		cur = itemsResp
	}
	nextURL := getODataCollectionNextPageURL(cur)
	if nextURL == "" {
		return res, nil
	}
	nextItemsResp, err := NewItems(items.client, nextURL, items.config).Get()
	if err != nil {
		return res, err
	}
	res = append(res, nextItemsResp.Data()...)
	return getAll(res, nextItemsResp, items)
}

/* Pagination helpers */

// ItemsPage - paged items
type ItemsPage struct {
	Items       ItemsResp
	HasNextPage func() bool
	GetNextPage func() (*ItemsPage, error)
}

// GetPaged gets Paged Items collection
func (items *Items) GetPaged() (*ItemsPage, error) {
	data, err := items.Get()
	if err != nil {
		return nil, err
	}
	res := &ItemsPage{
		Items: data,
		HasNextPage: func() bool {
			return data.HasNextPage()
		},
		GetNextPage: func() (*ItemsPage, error) {
			nextURL := data.NextPageURL()
			if nextURL == "" {
				return nil, fmt.Errorf("unable to get next page")
			}
			return NewItems(items.client, nextURL, items.config).GetPaged()
		},
	}
	return res, nil
}

// NextPageURL gets next page OData collection
func (itemsResp *ItemsResp) NextPageURL() string {
	return getODataCollectionNextPageURL(*itemsResp)
}

// HasNextPage returns is true if next page exists
func (itemsResp *ItemsResp) HasNextPage() bool {
	return itemsResp.NextPageURL() != ""
}
