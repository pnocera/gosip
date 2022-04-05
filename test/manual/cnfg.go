package manual

import (
	"fmt"

	"github.com/pnocera/gosip/auth/addin"
	u "github.com/pnocera/gosip/test/utils"
)

// ConfigReaderSpoAddinOnlyTest : test scenario
// noinspection GoUnusedExportedFunction
func ConfigReaderSpoAddinOnlyTest() {
	config := &addin.AuthCnfg{}
	err := config.ReadConfig(u.ResolveCnfgPath("./config/private.addin.json"))
	if err != nil {
		fmt.Printf("Error reading config: %v", err)
		return
	}
	fmt.Println(config)
}
