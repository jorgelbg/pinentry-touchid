# pinentry-touchid

<p align="center">
    <img class="center" src="https://user-images.githubusercontent.com/1291846/127916161-5803ca98-c0a2-4d1f-8479-860f4d7edc98.png" width="300" alt="pinentry-touchid logo"/>
</p>

Custom GPG pinentry program for macOS that allows using Touch ID for fetching the password from the
macOS keychain.

> Macbook Pro devices without Touch ID are currently not supported. These devices > lack a Touch ID
> sensor and while the alternative offered by Apple is to use (if available) an Apple Watch, this
> feature it is not yet implemented.

## See it in action

 ![pinentry-touchid in action with gopass](https://user-images.githubusercontent.com/1291846/128176593-271ac649-5207-41f2-83da-3fb3d37ede9c.gif)


## How does it work

This program interacts with the `gpg-agent` for providing a password, using the following rules:

- If the password entry for the given key cannot be found in the Keychain we fallback to the
  `pinentry-mac` program to get the password. We recommend preventing `pinentry-mac` from storing the
  password: uncheck the <kbd>Save in keychain</kbd> checkbox in the dialog.

- If a password entry is found the user will be shown the Touch ID dialog and upon successful
  authentication the password stored from the keychain will be returned to the gpg-agent.

- If a password entry is found but is not "owned" by the `pinentry-touchid` program after the
  successful authentication with Touch ID, a normal password will be shown. This is an extra step
  enforced by the macOS keychain. In this dialog click <kbd>Always allow</kbd> after entering the
  password. This will allow `pinentry-touchid` to access the password entry without the need to type
  the additional password, but still, the access to the password will be guarded by Touch ID.

## Installation

### Prerequisites

* [gnupg](https://formulae.brew.sh/formula/gnupg)
* [pinentry-mac](https://github.com/GPGTools/pinentry-mac)


If you have already installed GPG, make sure that executing `pinentry` shows a GUI prompt by running
the following command:

```sh
echo GETPIN | pinentry
```

You should get the dialog from [pinentry-mac](https://github.com/GPGTools/pinentry-mac). If that is not the case you can install it though Homebrew:

```sh
brew install pinentry-mac
```

You can overwrite the `pinentry` alias to point to `pinentry-mac`:

```sh
alias pinentry='pinentry-mac'
```

_Then try again whether you see a GUI prompt._

In some cases aliasing `pinentry` to `pinentry-mac` is not enough because `gpgconf` returns the
absolute path that points to the `$HOMEBREW_PREFIX/opt` path. In that case you can execute the
following command to automatically fix the symlink.

```sh
pinentry-touchid -fix
```

### Homebrew


As part of our release process we keep an updated Homebrew Formula. To install `pinentry-touchid` using
Homebrew execute the following commands:

```sh
brew tap jorgelbg/tap
brew install pinentry-touchid
```

Homebrew will print the next steps, which will look similar to:

```
==> Caveats
➡️  Ensure that pinentry-mac is the default pinentry program:
      /usr/local/bin/pinentry-touchid -fix

✅ Add the following line to your ~/.gnupg/gpg-agent.conf file:
      pinentry-program /usr/local/opt/pinentry-touchid/bin/pinentry-touchid

🔄  Then reload your gpg-agent:
      gpg-connect-agent reloadagent /bye

🔑  Run the following command to disable "Save in Keychain" in pinentry-mac:
    defaults write org.gpgtools.common DisableKeychain -bool yes

⛔️  If you are upgrading from a previous version, you will be asked to give
    access again to the keychain entry. Click "Always Allow" after the
    Touch ID verification to prevent this dialog from showing.
==> Summary
🍺  /usr/local/Cellar/pinentry-touchid/0.0.2: 4 files, 2.2MB, built in 10 seconds
```

### Manual installation

- Download the `pinentry-touchid` binary from our Releases page

- Configure the `gpg-agent` to use `pinentry-touchid` as its pinentry program. Add or replace the
  following line to your gpg agent configuration in: `~/.gnupg/gpg-agent.conf`:

```sh
pinentry-program /usr/local/bin/pinentry-touchid
```

You can replace `/usr/local/bin/pinentry-touchid` with the path where the binary was stored.

Make sure that the `pinentry-mac` is configured to be the default `pinentry` program (will be used
as fallback). You can check which PIN program will be used by default by executing:

```sh
pinentry-touchid -check
```

If any error is reported `pinentry-touchid` can automatically fix the symlink for you:
```sh
pinentry-touchid -fix
```

## Manually add your GPG key password to the Keychain

First, ensure pinentry-mac is already using the Keychain:

```sh
security find-generic-password -s 'GnuPG'
```

You should get a big list of attributes.
If you get an error, such as the following, it means pinentry-mac is not configured to use the Keychain:

```
security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain.
```

If you do not see this error, skip ahead to [Configuring pinentry-touchid](#configuring-pinentry-touchid).

### Configuring pinentry-mac

Before configuring pinentry-touchid, you should configure pinentry-mac to use the Keychain at least once:

```sh
defaults write org.gpgtools.common UseKeychain -bool yes
```

Note that there are two defaults which are the reverse of each other.
This one, `UseKeychain`, should be set to `yes` or `true`.

Ensure the `pinentry-program` entry in your `~/.gnupg/gpg-agent.conf` points to pinentry-mac, then restart the GPG Agent:

```sh
gpgconf --kill gpg-agent
```

Using gpg should then use pinentry-mac to provide a GUI prompt for your GPG passphrase:

```sh
echo 1234 | gpg -as -
```

Make sure you check the "Save in Keychain" box on the prompt.
You may then get a second prompt, this time for your login password, to authorize pinentry-mac to create and use the Keychain entry to store your GPG passphrase.
If so, use "Always Allow" to avoid future prompts.

You should now be able to see the new Keychain entry via the same command as before:

```sh
security find-generic-password -s 'GnuPG'
```

Continue on to the next section to replace this password prompt with a TouchID prompt.

### Configuring pinentry-touchid

Once your Keychain is configured correctly, you can update your `gpg-agent.conf` with the correct path for `pinentry-program` pointing to the full path to `pinentry-touchid`.
Remember to restart the GPG Agent each time you make a change to this configuration file:

```sh
gpgconf --kill gpg-agent
```

We recommend disabling the option to store the password in the macOS Keychain for the default
pinentry-mac program with the following option:

```sh
defaults write org.gpgtools.common DisableKeychain -bool yes
```

This will allow `pinentry-touchid` to create and automatically take ownership of the entry in the
Keychain. If an entry already exists in the Keychain you need to always allow `pinentry-touchid` to
access the existing entry.

## Disclaimer

This project does not store the password/pin in the [Secure
Enclave](https://support.apple.com/en-gb/guide/security/sec59b0b31ff/web) of your device, instead
uses the normal Keychain entry from
[pinentry-mac](https://github.com/GPGTools/pinentry/tree/master/macosx) if available, or creates a
new one.

## Tested on

I've tested `pinentry-touchid` in the following combinations of devices and macOS versions:

* MacBook Pro (15-inch, 2018), macOS Catalina - 10.15.7
* MacBook Pro (15-inch, 2018), macOS Big Sur - 11.4, 11.5.0, 11.5.1
* MacBook Pro (16-inch, Late 2019), macOS Big Sur - 11.4, 11.5.1
* MacBook Pro (16-inch, Late 2021), macOS Monterey - 12.2

## Links

* The project icon is taken from <a href="https://icons8.com/icon/BebbEec6QUjh/touch-id">Touch ID icon by Icons8</a>.
