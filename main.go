// Copyright (c) 2021 Jorge Luis Betancourt. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.
//
//go:build darwin && cgo
// +build darwin,cgo

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/enescakir/emoji"
	"github.com/foxcpp/go-assuan/common"
	"github.com/foxcpp/go-assuan/pinentry"
	pinentryBinary "github.com/gopasspw/pinentry"
	"github.com/jorgelbg/pinentry-touchid/sensor"
	"github.com/keybase/go-keychain"
	touchid "github.com/lox/go-touchid"
)

// AuthFunc is a function that runs some check to verify if the caller has access to the Keychain
// entry
type AuthFunc func(string) (bool, error)

// PromptFunc is a function that asks a password from the user
type PromptFunc func(pinentry.Settings) ([]byte, error)

// GetPinFunc is a function that executes the process for getting a password from the Keychain
type GetPinFunc func(pinentry.Settings) (string, *common.Error)

const (
	// DefaultLogFilename default name for the log files
	DefaultLogFilename = "pinentry-touchid.log"
	defaultLoggerFlags = log.Ldate | log.Ltime | log.Lshortfile
)

var (
	// DefaultLogLocation is the location of the log file
	DefaultLogLocation = filepath.Join(filepath.Clean(os.TempDir()), DefaultLogFilename)

	emailRegex = regexp.MustCompile(`\"(?P<name>.*<(?P<email>.*)>)\"`)
	keyIDRegex = regexp.MustCompile(`ID (?P<keyId>.*),`) // keyID should be of exactly 8 or 16 characters

	errEmptyResults    = errors.New("no matching entry was found")
	errMultipleMatches = errors.New("multiple entries matched the query")

	check      = flag.Bool("check", false, "Verify that pinentry-mac is present in the system.")
	fixSymlink = flag.Bool("fix", false, "Set up pinentry-mac as the fallback PIN entry program.")
	_          = flag.String("display", "", "Set the X display (unused)")
)

// checkEntryInKeychain executes a search in the current keychain. The search configured to not
// return the Data stored in the Keychain, as a result this should not require any type of
// authentication.
func checkEntryInKeychain(label string) (bool, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetLabel(label)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(false)
	query.SetReturnAttributes(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return false, err
	}

	return len(results) == 1, nil
}

// KeychainClient represents a single instance of a pinentry server
type KeychainClient struct {
	logger   *log.Logger
	authFn   AuthFunc
	promptFn PromptFunc
}

// New returns a new instance of KeychainClient with some sane defaults, a logger automatically
// configured, an authFn that invokes Touch ID and a promptFn that fallbacks to the pinentry-mac
// program.
func New() KeychainClient {
	var logger *log.Logger
	path := filepath.Clean(DefaultLogLocation)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			panic("Couldn't create log file")
		}

		logger = log.New(file, "", defaultLoggerFlags)
	} else {
		// append to the existing log file
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}

		logger = log.New(file, "", defaultLoggerFlags)
	}

	logger.Print("Ready!")

	return KeychainClient{
		logger:   logger,
		promptFn: passwordPrompt,
		authFn:   touchid.Authenticate,
	}
}

// WithLogger allows to create a new instance of KeychainClient with a custom logger
func WithLogger(logger *log.Logger) KeychainClient {
	return KeychainClient{
		logger:   logger,
		promptFn: passwordPrompt,
		authFn:   touchid.Authenticate,
	}
}

// passwordFromKeychain retrieves a password given a label from the Keychain
func passwordFromKeychain(label string) (string, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetLabel(label)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", errEmptyResults
	}

	if len(results) > 1 {
		return "", errMultipleMatches
	}

	return string(results[0].Data), nil
}

// storePasswordInKeychain saves a password/pin in the keychain with the given label
// and keyInfo
func storePasswordInKeychain(label, keyInfo string, pin []byte) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService("GnuPG")
	item.SetAccount(keyInfo)
	item.SetLabel(label)
	item.SetData(pin)
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)

	if err := keychain.AddItem(item); err != nil {
		return err
	}

	return nil
}

// passwordPrompt uses the default pinentry-mac program for getting the password from the user
func passwordPrompt(s pinentry.Settings) ([]byte, error) {
	p, err := pinentryBinary.New()
	if err != nil {
		return []byte{}, fmt.Errorf("failed to start %q: %w", pinentryBinary.GetBinary(), err)
	}
	defer p.Close()

	p.Set("title", "pinentry-touchid PIN Prompt")

	// passthrough the original description that its used for creating the keychain item
	p.Set("desc", strings.ReplaceAll(s.Desc, "\n", "\\n"))

	// Enable opt-in external PIN caching (in the OS keychain).
	// https://gist.github.com/mdeguzis/05d1f284f931223624834788da045c65#file-info-pinentry-L324
	//
	// Ideally if this option was not set, pinentry-mac should hide the `Save in Keychain`
	// checkbox, but this is not the case.
	// p.Option("allow-external-password-cache")
	p.Set("KEYINFO", s.KeyInfo)
	p.Set("PROMPT", s.Prompt)
	p.Set("REPEAT", s.RepeatPrompt)
	p.Set("REPEATERROR", s.RepeatError)

	return p.GetPin()
}

func assuanError(err error) *common.Error {
	return &common.Error{
		Src:     common.ErrSrcPinentry,
		SrcName: "pinentry",
		Code:    common.ErrCanceled,
		Message: err.Error(),
	}
}

// GetPIN executes the main logic for returning a password/pin back to the gpg-agent
func (c KeychainClient) GetPIN(s pinentry.Settings) (string, *common.Error) {
	if len(s.Error) == 0 && len(s.RepeatPrompt) == 0 && s.Opts.AllowExtPasswdCache && len(s.KeyInfo) != 0 {
		return GetPIN(c.authFn, c.promptFn, c.logger)(s)
	}

	// fallback to pinentry-mac in any other case
	pin, err := c.promptFn(s)
	if err != nil {
		return "", assuanError(err)
	}

	// TODO(jorge): try to persist automatically in the keychain?
	return string(pin), nil
}

// Confirm Asks for confirmation, not implemented.
func (c KeychainClient) Confirm(s pinentry.Settings) (bool, *common.Error) {
	c.logger.Println("Confirm was called!")

	if _, err := c.promptFn(s); err != nil {
		return false, assuanError(err)
	}

	return true, nil
}

// Msg shows a message, not implemented.
func (c KeychainClient) Msg(pinentry.Settings) *common.Error {
	c.logger.Println("Msg was called!")

	return nil
}

// GetPIN executes the main logic for returning a password/pin back to the gpg-agent
func GetPIN(authFn AuthFunc, promptFn PromptFunc, logger *log.Logger) GetPinFunc {
	return func(s pinentry.Settings) (string, *common.Error) {
		matches := emailRegex.FindStringSubmatch(s.Desc)
		name := strings.Split(matches[1], " <")[0]
		email := matches[2]

		matches = keyIDRegex.FindStringSubmatch(s.Desc)
		keyID := matches[1]

		// Drop the optional 0x prefix from keyID (--keyid-format)
		// https://www.gnupg.org/documentation/manuals/gnupg/GPG-Configuration-Options.html
		keyID = strings.TrimPrefix(keyID, "0x")

		if len(keyID) != 8 && len(keyID) != 16 {
			logger.Printf("Invalid keyID: %s", keyID)
			return "", assuanError(fmt.Errorf("invalid keyID: %s", keyID))
		}

		keychainLabel := fmt.Sprintf("%s <%s> (%s)", name, email, keyID)
		exists, err := checkEntryInKeychain(keychainLabel)
		if err != nil {
			logger.Printf("error checking entry in keychain: %s", err)
			return "", assuanError(err)
		}

		// If the entry is not found in the keychain, we trigger `pinentry-mac` with the option
		// to save the pin in the keychain.
		//
		// When trying to access the newly created keychain item we will get the normal password prompt
		// from the OS, we need to "Always allow" access to our application, still the access from our
		// app to the keychain item will be guarded by Touch ID.
		//
		// Currently I'm not aware of a way for automatically adding our binary to the list of always
		// allowed apps, see: https://github.com/keybase/go-keychain/issues/54.
		if !exists {
			pin, err := promptFn(s)
			if err != nil {
				logger.Printf("Error calling pinentry program (%s): %s", pinentryBinary.GetBinary(), err)
			}

			if len(pin) == 0 {
				logger.Printf("pinentry-mac didn't return a password")
				return "", assuanError(fmt.Errorf("pinentry-mac didn't return a password"))
			}

			// s.KeyInfo is always in the form of x/cacheId
			// https://gist.github.com/mdeguzis/05d1f284f931223624834788da045c65#file-info-pinentry-L357-L362
			keyInfo := strings.Split(s.KeyInfo, "/")[1]

			// pinentry-mac can create an item in the keychain, if that was the case, the user will have
			// to authorize our app to access the item without asking for a password from the user. If
			// not, we create an entry in the keychain, which automatically gives us ownership (i.e the
			// user will not be asked for a password). In either case, the access to the item will be
			// guarded by Touch ID.
			exists, err = checkEntryInKeychain(keychainLabel)
			if err != nil {
				logger.Printf("error checking entry in keychain: %s", err)
				return "", assuanError(err)
			}

			if !exists {
				// pinentry-mac didn't create a new entry in the keychain, we create our own and take
				// ownership over the entry.
				err = storePasswordInKeychain(keychainLabel, keyInfo, pin)

				if err == keychain.ErrorDuplicateItem {
					logger.Printf("Duplicated entry in the keychain")
					return "", assuanError(err)
				}
			} else {
				logger.Printf("The keychain entry was created by pinentry-mac. Permission will be required on next run.")
			}

			return string(pin), nil
		}

		var ok bool
		if ok, err = authFn(fmt.Sprintf("access the PIN for %s", keychainLabel)); err != nil {
			logger.Printf("Error authenticating with Touch ID: %s", err)
			return "", assuanError(err)
		}

		if !ok {
			logger.Printf("Failed to authenticate")
			return "", nil
		}

		password, err := passwordFromKeychain(keychainLabel)
		if err != nil {
			log.Printf("Error fetching password from Keychain %s", err)
		}

		return password, nil
	}
}

// validatePINBinary validates that the pinentry program returned by gpgconf
// is/points to pinentry-mac
func validatePINBinary() (string, error) {
	binaryPath := pinentryBinary.GetBinary()
	originalPath := binaryPath
	if _, err := exec.LookPath(binaryPath); err != nil {
		return binaryPath, errors.New("PIN entry program not found")
	}

	// check if the binary is (or resolves to -- if it is a symlink) to pinentry-mac
	info, err := os.Lstat(binaryPath)
	if err != nil {
		return binaryPath, fmt.Errorf("Couldn't lstat file: %w", err)
	}

	if info.Mode()&fs.ModeSymlink != 0 {
		// the file is a symlink so we check that the resolved path contains
		// pinentry-mac
		path, err := filepath.EvalSymlinks(binaryPath)
		if err != nil {
			return binaryPath, fmt.Errorf("Couldn't resolve symlink: %w", err)
		}

		binaryPath = path
	}
	if !strings.Contains(binaryPath, "pinentry-mac") {
		return "", errors.New(
			fmt.Sprintf("%s is a symlink that resolves to %s not to pinentry-mac",
				originalPath, binaryPath))
	}

	return binaryPath, nil
}

// fixPINBinary forces pinentry-mac as the fallback pinentry program
func fixPINBinary(oldPath string) error {
	newPath, err := exec.LookPath("pinentry-mac")
	if err != nil {
		return errors.New("pinentry-mac couldn't be found in your PATH")
	}

	if err := os.Remove(oldPath); err != nil {
		return fmt.Errorf("Unable to remove symlink: %w", err)
	}

	// create the new symlink pointing to pinentry-mac
	if err := os.Symlink(newPath, oldPath); err != nil {
		return fmt.Errorf("Unable to symlink to pinentry-mac: %w", err)
	}

	return nil
}

func main() {
	flag.Parse()
	if !sensor.IsTouchIDAvailable() {
		fmt.Fprintf(os.Stderr,
			"%v pinentry-touchid does not support devices without a Touch ID sensor!\n", emoji.CrossMark)
		os.Exit(-1)
	}

	if *fixSymlink {
		path := pinentryBinary.GetBinary()
		if err := fixPINBinary(path); err != nil {
			fmt.Fprintf(os.Stderr, "%v %s\n", emoji.CrossMark, err)
			os.Exit(-1)
		}

		fmt.Fprintf(os.Stderr, "%v %s is now pointing to pinentry-mac\n", emoji.CheckMarkButton, path)
		os.Exit(0)
	}

	if *check {
		path, err := validatePINBinary()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v %s %s\n", emoji.CrossMark, err, path)
			os.Exit(-1)
		}

		if path != "" {
			fmt.Fprintf(os.Stdout, "%v %s will be used as a fallback PIN program\n", emoji.CheckMarkButton, path)
		}

		os.Exit(0)
	}

	client := New()

	callbacks := pinentry.Callbacks{
		GetPIN:  client.GetPIN,
		Confirm: client.Confirm,
		Msg:     client.Msg,
	}

	if err := pinentry.Serve(callbacks, "Hi from pinentry-touchid!"); err != nil {
		fmt.Fprintf(os.Stderr, "Pinentry Serve returned error: %v\n", err)
		os.Exit(-1)
	}
}
