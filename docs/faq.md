# FAQ

## Why did Apple Terminal.app resize when displaying a QR Code?

When QR codes are displayed with `--show-qr` or `--demo-qr`, Apple Terminal
resizes to accommodate the QR codes.

Apple Terminal.app does not support displaying images in the terminal. The CLI
draws the QR code using solid blocks instead.

This QR code needs more space than the default `80x24` window. If the window is
too small, the CLI asks Terminal.app to make it larger before displaying the
QR code. When you press Enter or Ctrl+C, it restores the old window size.

The resize request is an xterm control sequence. For example, this command asks
the terminal to resize to 40 rows and 100 columns:

```
printf '\e[8;40;100t'
```

If the window cannot be resized, the CLI will show the required size. Try a
smaller font with `Command-`, use a [better terminal](#tested-terminals), or use
`--export-qr` instead.

Use `--debug` to see which QR display method was selected.

To test the resize behavior, start with a small Apple Terminal window and run:

```
twofa-rescue --demo-qr
```

To test several entries in one session:

```
TWOFA_RESCUE_PASS=test twofa-rescue --show-qr sample-encrypted.json
```

## Why is the QR code distorted over ssh from Apple Terminal.app?

ssh does not forward the `TERM_PROGRAM` environment variable by default. The
CLI cannot detect Apple Terminal.app without it, so it uses the generic Unicode
block renderer.

Use `--debug` to confirm it. You will see something like:

```
[debug] TERM_PROGRAM="" TERM="xterm-256color": no known image protocol, using Unicode block fallback
```

There is no reliable way to detect Apple Terminal.app from `TERM` alone.
To test this idea, you can set `TERM_PROGRAM` for the remote command:

```
ssh remote_host TERM_PROGRAM=Apple_Terminal twofa-rescue --demo-qr
```

Make sure the local Terminal window is large enough for the QR code. Otherwise,
use a [better terminal](#tested-terminals) or use `--export-qr`.


## How can I try the CLI without a real Ente Auth export?

Use `sample-encrypted.json` from the repo. The password is `test`:

```
TWOFA_RESCUE_PASS=test twofa-rescue sample-encrypted.json
```

The file contains fake accounts such as `alice@example.com`. To import only one into
an authenticator app using a QR code, use the filter `github`:

```
TWOFA_RESCUE_PASS=test twofa-rescue --show-qr sample-encrypted.json github
```

## Can I use the CLI without a real encrypted Ente Auth export?

Yes it's possible. If you know your 2FA secrets, you can create a text 
file like `sample-otpauth-urls.txt`, update account, issuer etc accordingly.
e.g.

```
otpauth://totp/GitHub:alice@example.com?secret=JBSWY3DPEHPK3PXP&issuer=GitHub
```

Then use the `--encrypt` option create a JSON file with encrypted secret.

Example:

```
TWOFA_RESCUE_PASS=yourpassword twofa-rescue --encrypt \
    -o myexport_encrypted.json plaintext_json.txt
```

## Why did Windows block the binary with "An Application Control policy has blocked this file"?

In File Explorer, right-click the binary > Properties > General tab.
If there's an "Unblock" checkbox near the bottom, check it and try running again.

On a personal Windows 11 system, this is likely Smart App Control (SAC), 
 which uses Microsoft's cloud reputation service and code signatures
to decide whether to trust an app. The Windows build of `twofa-rescue` isn't
currently signed, so a new or uncommon build can get flagged.

In my case, several freshly cross-compiled binaries were blocked at first. About
ten minutes after transferring another build, all of the previously-blocked
binaries started running without any changes on my end, likely a delayed
reputation check or a local security definition update. Waiting and retrying
may help, but it's not guaranteed.


## Why did I not abandon Ente Auth after the bug that motivated this CLI?

There are a few reasons. I am familiar with the crypto @ENTE_AUTH@ uses to
encrypt the exported JSON file. It uses Argon2id for key derivation and
libsodium's `crypto_secretstream_xchacha20poly1305` for encryption. I trust the
encrypted file enough to keep copies on all my systems. Of course, this 
assumes the export is protected with a strong passphrase.

I also live in terminals. If my phone is upstairs, I do not want to go get it
just for a 2FA code.

Most importantly, I can decrypt the export myself and import the secrets into
another Authenticator app whenever I want. I still use @ENTE_AUTH@, but I am no
longer dependent on it or any other single Authenticator app.
