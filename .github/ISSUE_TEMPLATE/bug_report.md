---
name: Bug report
about: Create a report to help us improve
title: ''
labels: bug
assignees: ''

---

## Describe the bug

A clear and concise description of what the bug is.

## System information

**macOS**
 - Architecture: (ARM/M1/Intel)
 - Version: (e.g. 11.6.1)

**GPG**
 - Output of `gpg --version`
 - Installed via Homebrew?

**Configuration**

 - Please attach the output of the command `gpgconf`.

**Logs**

**`gpg-agent`:**

It would be very useful for us if you could enable the `basic` debug info for your `gpg-agent` and attach the generated log. Add the following to your `~/.gpg-agent.conf`:

```
debug-level basic
log-file /Users/<USERNAME>/.gnupg/gpg-agent.log
```

Reload `gpg-agent` with the following command:
```sh
$ gpg-connect-agent reloadagent /bye
```

Add/attach the relevant section of the log to this issue (feel free to redact your key IDs).

**`pinentry-touchid`:**

`pinentry-touchid` also generates its own log which you can find in `/tmp/pinentry-touchid.log`.
