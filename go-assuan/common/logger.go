package common

import (
	"io/ioutil"
	"log"
)

// Logger used for *low-level* debug output by go-assuan.
// Redirected to ioutl.Discard by default.
var Logger log.Logger

func init() {
	Logger.SetPrefix("DEBUG(go-assuan/common): ")
	Logger.SetOutput(ioutil.Discard)
}
