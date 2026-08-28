# Features

## Display live 2FA codes

Can decrypt an @ENTE_AUTH@ encrypted export file and print the current 2FA
code for each entry.

**Set `TWOFA_RESCUE_PASS` env var with the password first.**  `twofa-rescue -h` for help.
```
 export TWOFA_RESCUE_PASS='your_secret'
twofa-rescue export_encrypted_json.json
```

Example:
```
$ twofa-rescue ente-auth-codes-2026-08-03.json
ISSUER  ACCOUNT            CODE    EXPIRES IN
======  =======            ====    ==========
GitHub  alice@example.com  123456  23s 14:32:07
AWS     alice@example.com  654321  59s 14:32:07
```
**Note:** Issuer, account, and codes shown above are placeholders, not real data


## Filtering

Pass extra words after the export file to narrow the results down to
entries whose issuer or account name matches.
```
twofa-rescue export_encrypted_json.json github
```

## Plaintext export (not recommended)

Decrypt the @ENTE_AUTH@ encrypted export file and print the raw `otpauth://` lines instead of
generating codes. Useful if you want to feed the plaintext into another
tool, or write it to a file with `-o`.
```
twofa-rescue --decrypt export_encrypted_json.json
twofa-rescue --decrypt -o plain.txt export_encrypted_json.json
```
**WARNING:** Be careful! It is not a good idea to create a plain text file of
the secrets on the disk. Use it only if there is a pressing need to do that.

Example output (issuers, accounts, and secrets shown below are placeholders,
not real data):
```
otpauth://totp/GitHub:alice@example.com?secret=JBSWY3DPEHPK3PXP&issuer=GitHub
otpauth://totp/AWS:alice@example.com?secret=KRSXG5CTMVRXEZLU&issuer=AWS
otpauth://totp/Google:alice.dev@example.com?secret=MFRGGZDFMZTWQ2LK&issuer=Google
otpauth://totp/Dropbox:alice@example.com?secret=NBSWY3DPO5XXE3DE&issuer=Dropbox
otpauth://totp/Cloudflare:alice.work@example.com?secret=ORSXG5BAMVQXG43F&issuer=Cloudflare
```

## Export 2FA QR Codes to PNG files (not recommended)

Write one QR code PNG per entry so you can move your accounts into
another authenticator app by scanning or uploading the image.
```
twofa-rescue --export-qr /path/to/dir export_encrypted_json.json
```
**WARNING:** These files contain your 2FA secrets in plaintext. Import them, then
delete the files and empty your trash. Do not leave them sitting on disk.

## QR codes in the terminal (recommended)

**NOTE:** Make sure your terminal can render a scannable QR code before using
this. See [Tested Terminals](#tested-terminals) for details.

**NOTE:** On terminals without an inline image protocol, the QR code is
drawn with text block characters instead of a real image. This only looks
square if the terminal's character cell is close to twice as tall as it
is wide, which is not always the case. On Apple's Terminal.app, this can
make the QR code look squished. If that happens, use `--export-qr`
instead, or switch to a terminal with inline image support such as
@ITERM2@.

Display the QR codes one at a time right in the terminal instead of
writing PNG files to disk. Press Enter to move to the next one and
Ctrl+C to quit.

This method can be used to import 2FA secrets into most 2FA authenticator apps.

```
twofa-rescue --show-qr export_encrypted_json.txt
```

## Screenshots of demo QR code

### iTerm2

Show a safe, non-sensitive QR code without needing an export file or a
password to test if the image is recognized by other 2FA mobile apps.
Here is a screenshot of @ITERM2@ displaying a demo QR Code on the terminal:

These [Terminals](#tested-terminals) can display QR code successfully as well.
Apple Terminal has a limitation, read below.

```
twofa-rescue --demo-qr
```

<table>
  <tr>
    <td align="center">
      <img src="images/demo_totp.png" alt="Demo QR Code iTerm2" /><br />
      <sub>Demo QR code on iTerm2</sub>
    </td>
  </tr>
</table>

### Apple Terminal

This is a test on a junk 2012 13" MacBook Pro with Apple Terminal

<p align="center">
    <img
        src="images/macbookpro.png"
        width="40%"
        alt="2012 old macbook pro"
    />
</p>

<table>
  <tr>
    <td align="center">
      <img src="images/apple_terminal1.png" alt="Demo QR Code Apple Terminal" /><br />
      <sub>Demo QR code on an Apple Terminal, QR Code display failed</sub>
    </td>
  </tr>
</table>

- Above: with default font size, you will see the window will resize but will fail to display the QR Code. Note: this will not happen if you've a big enough screen.

<table>
  <tr>
    <td align="center">
      <img src="images/apple_terminal2.png" alt="Demo QR Code Apple Terminal" /><br />
      <sub>Demo QR code on on a an 13" macbook pro Apple Terminal</sub>
    </td>
  </tr>
</table>

- Notice Above: had to reduced font size 3 times by typing `Command-` and after that
the QR code was displayed successfully. Look at [FAQ](#faq) for details on
Apple Terminal limitations.

<table>
  <tr>
    <td align="center">
      <img src="images/apple_terminal3.png" alt="Demo QR Code Apple Terminal" /><br />
      <sub>Demo QR code on on a an 13" macbook pro Apple Terminal</sub>
    </td>
  </tr>
</table>

- Expanded window with QR code


### Windows PowerShell

<table>
  <tr>
    <td align="center">
      <img src="images/windows11.png" alt="Demo QR Code Windows 11" /><br />
      <sub>Demo QR code on on a Windows 11 Terminal</sub>
    </td>
  </tr>
</table>

Try any mobile 2FA app to test that 2FA secrets can be imported by scanning
the QR Code on the terminsl. I tested the following authenticator apps on iOS v26.6:

* @GOOGLE_AUTHENTICATOR@
* @AUTHENTICATOR@
* @AUTHY@
* @ENTE_AUTH@ on iOS

## Show 2FA secret QR Code

Example on how to display QR Code with 2FA secrets. The displayed QR Code can
be used to import to other Authenticator Apps by scanning it with your camera
of your mobile device. 

Here it  is taking a sample encrypted @ENTE_AUTH@ export JSON file as input. 
Example:

<table>
  <tr>
    <td align="center">
      <img src="images/show_qr1.png" alt="Show 2FA secret1" /><br />
      <sub>Command to display QRCode with 2FA secret</sub>
    </td>
  </tr>
</table>

<table>
  <tr>
    <td align="center">
      <img src="images/show_qr2.png" alt="Show 2FA secret2" /><br />
      <sub>Display QR code of 2FA secret with code one by one. Can be imported by pointing camera of your Authenticator app</sub>
    </td>
  </tr>
</table>

The CLI uses a pure @GO@ implementation compatible with @LIBSODIUM@'s
secretstream format for encryption and decryption.
