package pinentry

import (
	"io/ioutil"
	"log"
)

// Logger used for *high-level pinentry* debug output by go-assuan.
// Redirected to ioutl.Discard by default.
var Logger log.Logger

func init() {
	Logger.SetPrefix("DEBUG(go-assuan/pinentry): ")
	Logger.SetOutput(ioutil.Discard)
}
