package manual

import (
	"fmt"
	"strings"

	"github.com/pnocera/gosip"
	"github.com/pnocera/gosip/api"
)

// CheckBasicPost : try creating an item
// noinspection GoUnusedExportedFunction
func CheckBasicPost(client *gosip.SPClient) (string, error) {
	sp := api.NewHTTPClient(client)
	endpoint := client.AuthCnfg.GetSiteURL() + "/_api/web/lists/getByTitle('Custom')/items"
	body := `{"__metadata":{"type":"SP.Data.CustomListItem"},"Title":"Test"}`

	data, err := sp.Post(endpoint, strings.NewReader(body), nil)
	if err != nil {
		return "", fmt.Errorf("unable to read a response: %w", err)
	}

	return string(data), nil
}
