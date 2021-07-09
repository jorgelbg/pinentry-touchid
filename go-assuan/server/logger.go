package server

import (
	"io/ioutil"
	"log"
)

// Logger used for *server-side* debug output by go-assuan.
// Redirected to ioutl.Discard by default.
var Logger log.Logger

func init() {
	Logger.SetPrefix("DEBUG(go-assuan/server): ")
	Logger.SetOutput(ioutil.Discard)
}
