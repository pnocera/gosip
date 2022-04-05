// Package api represents Fluent API for SharePoint object model
package api

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pnocera/gosip"
)

var storage = cache.New(5*time.Minute, 10*time.Minute)

//go:generate ggen -ent SP -conf

// SP represents SharePoint REST+ API root struct
// Always use NewSP constructor instead of &SP{}
type SP struct {
	client *gosip.SPClient
	config *RequestConfig
}

// NewSP - SP struct constructor function
func NewSP(client *gosip.SPClient) *SP {
	return &SP{client: client}
}

// ToURL gets endpoint with modificators raw URL
func (sp *SP) ToURL() string {
	return sp.client.AuthCnfg.GetSiteURL()
}

// Web API object getter
func (sp *SP) Web() *Web {
	return NewWeb(
		sp.client,
		fmt.Sprintf("%s/_api/Web", sp.ToURL()),
		sp.config,
	)
}

// Site API object getter
func (sp *SP) Site() *Site {
	return NewSite(
		sp.client,
		fmt.Sprintf("%s/_api/Site", sp.ToURL()),
		sp.config,
	)
}

// Utility getter
func (sp *SP) Utility() *Utility {
	return NewUtility(sp.client, sp.ToURL(), sp.config)
}

// Search getter
func (sp *SP) Search() *Search {
	return NewSearch(
		sp.client,
		fmt.Sprintf("%s/_api/Search", sp.ToURL()),
		sp.config,
	)
}

// Profiles getter
func (sp *SP) Profiles() *Profiles {
	return NewProfiles(
		sp.client,
		fmt.Sprintf("%s/_api/sp.userprofiles.peoplemanager", sp.ToURL()),
		sp.config,
	)
}

// Taxonomy getter
func (sp *SP) Taxonomy() *Taxonomy {
	return NewTaxonomy(sp.client, sp.ToURL(), sp.config)
}

// ContextInfo gets current Context Info object data
func (sp *SP) ContextInfo() (*ContextInfo, error) {
	return NewContext(sp.client, sp.ToURL(), sp.config).Get()
}

// Metadata returns $metadata info
func (sp *SP) Metadata() ([]byte, error) {
	client := NewHTTPClient(sp.client)
	return client.Get(
		fmt.Sprintf("%s/_api/$metadata", sp.ToURL()),
		sp.config,
	)
}
