package main

import (
	"fmt"

	"github.com/koltyakov/gosip/api"
	m "github.com/koltyakov/gosip/test/manual"
)

func main() {
	// client := m.GetAdfsAuthTest()
	// client := m.GetWapAuthTest()
	client := m.GetWapAdfsAuthTest()
	// client := m.GetOnlineADFSTest()
	// client := m.GetNtlmAuthTest()
	// client := m.GetAddinAuthTest()
	// client := m.GetFbaAuthTest()
	// client := m.GetSamlAuthTest()
	// client := m.GetTmgAuthTest()

	if client == nil {
		fmt.Println("No client")
	}

	// resp, err := m.CheckBasicPost(client)
	// if err != nil {
	// 	fmt.Printf("error in CheckBasicPost: %v\n", err)
	// }
	// fmt.Printf("response from CheckBasicPost: %s\n", resp)

	sp := api.SP{SPClient: client}
	res, err := sp.Web().Select("Title").Get()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%s\n", res)

}
