# pinentry-touchid

Custom pinentry program for macOS that allows to use Touch ID for fetching the password from the
macOS keychain.

We recommend to disable the option to store the password in the macOS Keychain with the following
option:

```sh
$ defaults write org.gpgtools.common DisableKeychain -bool yes
```

This will allow `pinentry-touchid` to create and automatically take ownership of the entry in the
Keychain. If an entry already exist in the Keychain you need to allow `pinentry-touchid` to always
access the existing entry.

We don't use the