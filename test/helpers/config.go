package helpers

import (
	"os"

	u "github.com/pnocera/gosip/test/utils"
)

// ConfigExists : checks if the config file exists
func ConfigExists(cnfgPath string) bool {
	// fmt.Printf("Checking config: %s", cnfgPath)
	if _, err := os.Stat(u.ResolveCnfgPath(cnfgPath)); err != nil {
		if os.IsNotExist(err) {
			// fmt.Printf(" | Not found, skipped.\n")
			return false
		}
	}
	// fmt.Printf("\n")
	return true
}
