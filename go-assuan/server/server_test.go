package server_test

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/foxcpp/go-assuan/common"
	"github.com/foxcpp/go-assuan/server"
)

type State struct {
	desc string
}

func setdesc(_ io.ReadWriter, state interface{}, params string) *common.Error {
	state.(*State).desc = params
	return nil
}

func getpin(pipe io.ReadWriter, state interface{}, _ string) *common.Error {
	s := bufio.NewScanner(os.Stdout)
	fmt.Println(state.(*State).desc)
	fmt.Print("Enter PIN: ")
	if ok := s.Scan(); !ok {
		return &common.Error{
			common.ErrSrcUnknown, common.ErrGeneral,
			"system", "I/O error",
		}
	}
	common.WriteData(pipe, s.Bytes())
	return nil
}

func ExampleProtoInfo() {
	pinentry := server.ProtoInfo{
		Greeting: "Pleased to meet you",
		Handlers: map[string]server.CommandHandler{
			"SETDESC": server.CommandHandler(setdesc),
			"GETPIN":  server.CommandHandler(getpin),
		},
		Help: map[string][]string{
			"SETDESC": {
				"Set request description",
			},
			"GETPIN": {
				"Read string from TTY",
			},
		},
		GetDefaultState: func() interface{} {
			return &State{"default desc"}
		},
	}
	server.ServeStdin(pinentry)
}
