package helpers

import (
	"context"
	"fmt"

	"github.com/pnocera/gosip"
	u "github.com/pnocera/gosip/test/utils"
)

// CheckDigest : check getting form digest
func CheckDigest(auth gosip.AuthCnfg, cnfgPath string) error {
	err := auth.ReadConfig(u.ResolveCnfgPath(cnfgPath))
	if err != nil {
		return err
	}

	client := &gosip.SPClient{
		AuthCnfg: auth,
	}

	digest, err := gosip.GetDigest(context.Background(), client)
	if err != nil {
		return fmt.Errorf("unable to get digest: %w", err)
	}

	if digest == "" {
		return fmt.Errorf("got empty digest")
	}

	if _, err := gosip.GetDigest(context.Background(), client); err != nil {
		return fmt.Errorf("unable to get cached digest: %w", err)
	}

	// if digest != cachedDigest {
	// 	return fmt.Errorf("digest cache is broken")
	// }

	return nil
}
