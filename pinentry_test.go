package main

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/foxcpp/go-assuan/pinentry"
	"github.com/keybase/go-keychain"
)

const (
	emptyPassword = ""
	testPassword  = "toomanysecrets2"
	keyDesc       = `Please enter the passphrase to unlock the OpenPGP secret key:
"Firstname Lastname <test@email.com>"
2048-bit RSA key, ID 61AF059BD632F971,
created 2021-01-01 (main key ID 70D56DF4CA30DE16).
`
	keyInfo = "n/8043823CBC5C5A0C66866520F333076D"
)

func TestStoreEntryInKeychain(t *testing.T) {
	err := storePasswordInKeychain("sampleLabel", "keyInfo", []byte(testPassword))

	if err != nil {
		t.Fatalf("storing entry in the Keychain should succeed: %s", err)
	}
}

func TestGetPasswordFromKeychain(t *testing.T) {
	defer func() {
		err := cleanKeychain("sampleLabel")

		if err != nil {
			t.Fatalf("failed to clear entry from Keychain: %s", err)
		}
	}()

	pass, err := passwordFromKeychain("sampleLabel")

	if err != nil {
		t.Fatalf("fetch entry from Keychain should succeed: %s", err)
	}

	if pass != testPassword {
		t.Fatalf("password mismatch got: %s want: %s", pass, testPassword)
	}
}

func TestGetPINSuccessfulAuthentication(t *testing.T) {
	keychainLabel := `Firstname Lastname <test@email.com> (61AF059BD632F971)`
	defer func() { _ = cleanKeychain(keychainLabel) }()

	params := pinentry.Settings{
		Desc:    keyDesc,
		KeyInfo: keyInfo,
	}

	err := storePasswordInKeychain(keychainLabel, keyInfo, []byte(testPassword))
	if err != nil {
		t.Fatalf("failed precreating entry in the Keychain: %s", err)
	}

	logger := &log.Logger{}
	logger.SetOutput(ioutil.Discard)

	fn := GetPIN(func(reason string) (bool, error) { return true, nil }, logger)
	pass, pinErr := fn(params)

	if pinErr != nil {
		t.Fatalf("call to GetPIN should succeed: %s", err)
	}

	if pass != testPassword {
		t.Fatalf("password mismatch got: %s want: %s", pass, testPassword)
	}
}

func TestGetPINUnsuccessfulAuthentication(t *testing.T) {
	keychainLabel := `Firstname Lastname <test@email.com> (61AF059BD632F971)`
	defer func() { _ = cleanKeychain(keychainLabel) }()

	logger := log.New(ioutil.Discard, "", 0)

	params := pinentry.Settings{
		Desc:    keyDesc,
		KeyInfo: keyInfo,
	}

	err := storePasswordInKeychain(keychainLabel, keyInfo, []byte(testPassword))
	if err != nil {
		t.Fatalf("failed precreating entry in the Keychain: %s", err)
	}

	fn := GetPIN(func(reason string) (bool, error) { return false, nil }, logger)
	pass, pinErr := fn(params)

	if pinErr != nil {
		t.Fatalf("call to GetPIN should succeed: %s", err)
	}

	if pass != emptyPassword {
		t.Fatalf("password mismatch got: %s want: %s", pass, testPassword)
	}
}

// Removes a matching entry from the "main" keychain.
// Since the item gets added in the same process, i.e temporal build while executing the test, it
// shouldn't request the password from the user.
func cleanKeychain(label string) error {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetLabel(label)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)

	return keychain.DeleteItem(query)
}
