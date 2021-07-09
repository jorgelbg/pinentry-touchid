package client

import (
	"io/ioutil"
	"log"
)

// Logger used for *client-side* debug output by go-assuan.
// Redirected to ioutl.Discard by default.
var Logger log.Logger

func init() {
	Logger.SetPrefix("DEBUG(go-assuan/client): ")
	Logger.SetOutput(ioutil.Discard)
}
