# Usage

Set the @ENTE_AUTH@ export password in `TWOFA_RESCUE_PASS` first:
```
  export TWOFA_RESCUE_PASS='password'
```

Tip: if your shell has `HISTCONTROL=ignorespace` set (the default in many
bash setups), type a single leading space before the `export` command to
keep the password out of your shell history.

<!-- BEGIN GENERATED USAGE: DO NOT EDIT BY HAND, run scripts/gen_usage.sh -->
```
Version: @($) twofa-rescue v1.0.1
https://github.com/muquit/twofa-rescue
Compiled with go version: go1.27.0

Usage: twofa-rescue [options] [export-file] [filter]

Arguments:
  [export-file]  Ente Auth encrypted export JSON
  [filter]       Optional issuer or account substring

Options:
  -h, --help         Show help
  -v, --version      Show version
  --decrypt          Print decrypted data instead of codes
  --encrypt          Encrypt any file as an Ente Auth export
  -o <file>          Write --decrypt or --encrypt output to a file
  --export-qr <dir>  Save one QR-code PNG per matching entry
  --show-qr          Show matching QR codes one at a time in the terminal
  --demo-qr          Show a non-sensitive test QR code in the terminal
  --debug            Print QR terminal-detection details to stderr

WARNING: QR-code PNGs contain plaintext 2FA secrets. Import them, delete
them, and empty your trash.

Environment:
  TWOFA_RESCUE_PASS  Password for encryption and export operations

  Linux/macOS (bash/zsh):
    export TWOFA_RESCUE_PASS='your-password'

  Windows (cmd.exe):
    set TWOFA_RESCUE_PASS=your-password

  Windows (PowerShell):
    $env:TWOFA_RESCUE_PASS='your-password'

Examples:
  twofa-rescue export_encrypted_json.txt
  twofa-rescue export_encrypted_json.txt github
  twofa-rescue --decrypt -o plain.txt export_encrypted_json.txt
  twofa-rescue --encrypt -o encrypted.json plain.txt
  twofa-rescue --export-qr /path/to/dir export_encrypted_json.txt
  twofa-rescue --show-qr export_encrypted_json.txt
  twofa-rescue --demo-qr

Note: flags must precede the file argument.
```
<!-- END GENERATED USAGE -->

If no filter is specified, codes for all entries are displayed. See
[Display live 2FA codes](#display-live-2fa-codes) for example.

## Options description

| Option | Description |
|---|---|
| `-h`, `--help` | Show this help message and exit. |
| `-v`, `--version` | Print the version and exit. |
| `--decrypt` | Decrypt the export and print the plaintext (`otpauth://` lines) instead of generating codes. |
| `-o <file>` | With `--decrypt`, write the decrypted plaintext to `<file>` instead of stdout. With `--encrypt`, write the encrypted file to `<file>` instead of stdout. |
| `--encrypt` | Encrypt an input file, text or binary (given in place of `<export-file>`), into @ENTE_AUTH@'s JSON export format, using the same Argon2id parameters @ENTE_AUTH@'s own app uses. Works on any file, not just `otpauth://` lines. |
| `--export-qr <dir>` | Write one QR-code PNG per entry (optionally narrowed by `[filter]`) into `<dir>`, importable one at a time into any authenticator app that supports QR/image upload. **WARNING:** These PNGs contain your 2FA secrets in plaintext. Import them, then delete the files (and empty your trash); do not leave them on disk. |
| `--show-qr` | Display QR codes one at a time in the terminal (optionally narrowed by `[filter]`). Press Enter to advance to the next, or Ctrl+C to quit. Nothing is written to disk. |
| `--demo-qr` | Display a non-sensitive test QR code in the terminal. No export file or password is required. |
| `--debug` | Print terminal-detection diagnostics to stderr when used with `--show-qr` or `--demo-qr`, showing which QR rendering path was chosen and why. |
