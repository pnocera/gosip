package helpers

import (
	"fmt"
	"reflect"
	"time"

	"github.com/pnocera/gosip"
	u "github.com/pnocera/gosip/test/utils"
)

// CheckAuth : common test case
func CheckAuth(auth gosip.AuthCnfg, cnfgPath string, required []string) error {
	err := auth.ReadConfig(u.ResolveCnfgPath(cnfgPath))
	if err != nil {
		return err
	}

	if err := CheckAuthProps(auth, required); err != nil {
		return err
	}

	if auth.GetStrategy() == "ntlm" || auth.GetStrategy() == "anonymous" {
		return nil
	}

	token, _, err := auth.GetAuth()
	if err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("accessToken is blank")
	}

	// Second auth should involve caching and be instant
	startAt := time.Now()
	token, _, err = auth.GetAuth()
	if err != nil {
		return err
	}
	if time.Since(startAt).Seconds() > 0.001 {
		return fmt.Errorf("possible caching issue, too slow read: %f", time.Since(startAt).Seconds())
	}
	if token == "" {
		return fmt.Errorf("accessToken is blank")
	}

	return nil
}

// CheckAuthProps : checks if all required props are provided
func CheckAuthProps(auth gosip.AuthCnfg, required []string) error {
	var missedProps []string
	for _, prop := range required {
		v := getPropVal(auth, prop)
		if v == "" {
			// return fmt.Errorf("doesn't contain required property value: %s", prop)
			missedProps = append(missedProps, prop)
		}
	}
	if len(missedProps) == 1 {
		return fmt.Errorf("doesn't contain required property value: %s", missedProps[0])
	}
	if len(missedProps) > 1 {
		return fmt.Errorf("doesn't contain required properties: %+v", missedProps)
	}
	return nil
}

func getPropVal(v gosip.AuthCnfg, field string) string {
	r := reflect.ValueOf(v)
	f := reflect.Indirect(r).FieldByName(field)
	if !f.IsValid() {
		return ""
	}
	return f.String()
}
