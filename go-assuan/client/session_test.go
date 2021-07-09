package client_test

import (
	"fmt"
	"net"

	assuan "github.com/foxcpp/go-assuan/client"
)

func ExampleSession() {
	// Connect to dirmngr.
	conn, _ := net.Dial("unix", ".gnupg/S.dirmngr")
	ses, _ := assuan.Init(conn)
	defer ses.Close()

	// Search for my key on default keyserver.
	data, _ := ses.SimpleCmd("KS_SEARCH", "foxcpp")
	fmt.Println(string(data))
	// data []byte = "info:1:1%0Apub:2499BEB8B47B0235009A5F0AEE8384B0561A25AF:..."

	// More complex transaction: send key to keyserver.
	ses.Transact("KS_PUT", "", map[string][]byte{
		"KEYBLOCK":      {},
		"KEYBLOCK_INFO": {},
	})
}
