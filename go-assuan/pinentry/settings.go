package pinentry

import "time"

type Options struct {
	Grab                bool
	AllowExtPasswdCache bool
	Display             string
	TTYType             string
	TTYName             string
	TTYAlert            string
	LCCtype             string
	LCMessages          string
	Owner               string
	TouchFile           string
	ParentWID           string
	InvisibleChar       string
}

// Settings struct contains options for pinentry prompt.
type Settings struct {
	// Detailed description of request.
	Desc string
	// Text right before textbox.
	Prompt string
	// Error to show. Reset after GetPin.
	Error string
	// Text on OK button.
	OkBtn string
	// Text on NOT OK button.
	// Broken in GnuPG's pinentry (2.2.5).
	NotOkBtn string
	// Text on CANCEL button.
	CancelBtn string
	// Window title.
	Title string
	// Prompt timeout. Any user interaction disables timeout.
	Timeout time.Duration
	// Text right before repeat textbox.
	// Repeat textbox is hidden after GetPin.
	RepeatPrompt string
	// Error text to be shown if passwords do not match.
	RepeatError string
	// Text before password quality bar.
	QualityBar string
	// Password quality callback.
	PasswordQuality func(string) int
	// Information from the key
	KeyInfo string

	Opts Options
}
