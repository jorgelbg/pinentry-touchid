package server

import (
	"bufio"
	"io"

	"github.com/foxcpp/go-assuan/common"
)

// Inquire requests data with specified keywords from client.
//
// It's better to explain by example:
//  Inquire(pipe, []byte{"KEYBLOCK", "KEYBLOCK_INFO"})
//
//  Will result in following messages sent by peers (S - server, C - client):
//  S: INQUIRE KEYBLOCK
//  C: D ...
//  C: END
//  S: INQUIRE KEYBLOCK_INFO
//  C: D ...
//  C: END
//
// Note: No OK or ERR sent after completion. You must report errors returned
// by this function manually using WriteError or send OK.
// This function can return common.Error, so you can do the following:
//	 data, err := server.Inquire(scnr, pipe, ...)
//	 if err != nil {
//	     if e, ok := err.(common.Error); ok {
//	  	   // Protocol error, report it to other peer (client).
//	         common.WriteError(pipe, e)
//	     } else {
//	  	   // Internal error, do something else...
//	     }
//	 }
func Inquire(scnr *bufio.Scanner, pipe io.Writer, keywords []string) (res map[string][]byte, err error) {
	Logger.Println("Sending inquire group:", keywords)
	for _, keyword := range keywords {
		if err := common.WriteLine(pipe, "INQUIRE", keyword); err != nil {
			Logger.Println("... I/O error:", err)
			return nil, err
		}

		data, err := common.ReadData(scnr)
		if err != nil {
			Logger.Println("... I/O error:", err)
			return nil, err
		}

		res[keyword] = data
	}
	return res, nil
}
