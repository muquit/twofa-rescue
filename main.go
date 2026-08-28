/*
License is MIT

Copyright © 2026 https://muquit.com/

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the "Software"),
to deal in the Software without restriction, including without limitation
the rights to use, copy, modify, merge, publish, distribute, sublicense,
and/or sell copies of the Software, and to permit persons to whom the
Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included
in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// Set at build time with -ldflags "-X main.version=x.y.z".
var (
	me         = "twofa-rescue"
	version    = "dev"
	projectUrl = "https://github.com/muquit/twofa-rescue"
)

// Used by qr.go to print terminal detection details.
var debugMode bool

type Export struct {
	Version         int    `json:"version"`
	KDFParams       KDF    `json:"kdfParams"`
	EncryptedData   string `json:"encryptedData"`
	EncryptionNonce string `json:"encryptionNonce"`
}

type KDF struct {
	MemLimit int    `json:"memLimit"`
	OpsLimit int    `json:"opsLimit"`
	Salt     string `json:"salt"`
}

// Decrypt an Ente Auth export in memory.
func decryptExport(path, password string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading export file: %w", err)
	}

	var export Export
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("parsing export JSON: %w", err)
	}
	if export.Version != 1 {
		return nil, fmt.Errorf("unsupported export version %d", export.Version)
	}

	encryptedData, err := base64.StdEncoding.DecodeString(export.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("decoding encryptedData: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(export.EncryptionNonce)
	if err != nil {
		return nil, fmt.Errorf("decoding encryptionNonce: %w", err)
	}

	key, err := deriveArgonKey(password, export.KDFParams.Salt, export.KDFParams.MemLimit, export.KDFParams.OpsLimit)
	if err != nil {
		return nil, fmt.Errorf("deriving key: %w", err)
	}

	plaintext, err := decryptChaCha20poly1305(encryptedData, key, nonce)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}
	return plaintext, nil
}

// Same encryption defaults used by Ente Auth.
const (
	encryptSaltBytes     = 16
	encryptMemLimitBytes = 1073741824 // 1 GiB
	encryptOpsLimit      = 4
)

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generating random bytes: %w", err)
	}
	return b, nil
}

// Encrypt data in the JSON format used by Ente Auth.
func encryptToExport(plaintext []byte, password string) ([]byte, error) {
	salt, err := randomBytes(encryptSaltBytes)
	if err != nil {
		return nil, err
	}
	saltB64 := base64.StdEncoding.EncodeToString(salt)

	key, err := deriveArgonKey(password, saltB64, encryptMemLimitBytes, encryptOpsLimit)
	if err != nil {
		return nil, fmt.Errorf("deriving key: %w", err)
	}

	header, err := randomBytes(secretStreamHeaderBytes)
	if err != nil {
		return nil, err
	}

	ciphertext, err := encryptChaCha20poly1305(plaintext, key, header)
	if err != nil {
		return nil, fmt.Errorf("encrypting: %w", err)
	}

	export := Export{
		Version: 1,
		KDFParams: KDF{
			MemLimit: encryptMemLimitBytes,
			OpsLimit: encryptOpsLimit,
			Salt:     saltB64,
		},
		EncryptedData:   base64.StdEncoding.EncodeToString(ciphertext),
		EncryptionNonce: base64.StdEncoding.EncodeToString(header),
	}

	data, err := json.Marshal(export)
	if err != nil {
		return nil, fmt.Errorf("marshaling export JSON: %w", err)
	}
	return data, nil
}

type entry struct {
	issuer  string
	account string
	key     *otp.Key
}

const demoOTPURI = "otpauth://totp/twofa-rescue%20Demo:demo@example.com?secret=JBSWY3DPEHPK3PXP&issuer=twofa-rescue%20Demo"

const passwordEnv = "TWOFA_RESCUE_PASS"

func demoEntry() (entry, error) {
	key, err := otp.NewKeyFromURL(demoOTPURI)
	if err != nil {
		return entry{}, fmt.Errorf("creating demo entry: %w", err)
	}
	return entry{issuer: key.Issuer(), account: key.AccountName(), key: key}, nil
}

func loadEntries(plaintext []byte) ([]entry, error) {
	var entries []entry
	scanner := bufio.NewScanner(bytes.NewReader(plaintext))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "otpauth://") {
			continue
		}
		key, err := otp.NewKeyFromURL(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping unparseable entry: %v\n", err)
			continue
		}
		entries = append(entries, entry{
			issuer:  key.Issuer(),
			account: key.AccountName(),
			key:     key,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if !strings.EqualFold(entries[i].issuer, entries[j].issuer) {
			return strings.ToLower(entries[i].issuer) < strings.ToLower(entries[j].issuer)
		}
		return strings.ToLower(entries[i].account) < strings.ToLower(entries[j].account)
	})
	return entries, nil
}

func generateCode(e entry) (code string, remaining int, err error) {
	period := int(e.key.Period())
	if period == 0 {
		period = 30
	}
	code, err = totp.GenerateCodeCustom(e.key.Secret(), time.Now(), totp.ValidateOpts{
		Period:    uint(period),
		Digits:    e.key.Digits(),
		Algorithm: e.key.Algorithm(),
	})
	if err != nil {
		return "", 0, err
	}
	remaining = period - int(time.Now().Unix())%period
	return code, remaining, nil
}

func formatCode(code string) string {
	if len(code) <= 3 {
		return code
	}
	return code[:3] + " " + code[3:]
}

func matches(e entry, filter string) bool {
	if filter == "" {
		return true
	}
	f := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(e.issuer), f) ||
		strings.Contains(strings.ToLower(e.account), f)
}

func printUsage() {
	fmt.Printf("Version: @($) %s %s\n", me, version)
	fmt.Printf("%s\n", projectUrl)
	compiledWith := "Compiled with go version: " + runtime.Version()
	fmt.Printf("%s\n\n", compiledWith)
	fmt.Printf(`Usage: %s [options] [export-file] [filter]

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
`, filepath.Base(os.Args[0]))

}

func main() {
	var (
		showHelp    bool
		showVersion bool
		decryptOnly bool
		doEncrypt   bool
		outFile     string
		exportQRDir string
		showQR      bool
		demoQR      bool
	)

	fs := flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ExitOnError)
	fs.BoolVar(&showHelp, "h", false, "")
	fs.BoolVar(&showHelp, "help", false, "")
	fs.BoolVar(&showVersion, "v", false, "")
	fs.BoolVar(&showVersion, "version", false, "")
	fs.BoolVar(&decryptOnly, "decrypt", false, "")
	fs.BoolVar(&doEncrypt, "encrypt", false, "")
	fs.StringVar(&outFile, "o", "", "")
	fs.StringVar(&exportQRDir, "export-qr", "", "")
	fs.BoolVar(&showQR, "show-qr", false, "")
	fs.BoolVar(&demoQR, "demo-qr", false, "")
	fs.BoolVar(&debugMode, "debug", false, "")
	fs.Usage = printUsage
	fs.Parse(os.Args[1:])

	if showHelp {
		printUsage()
		return
	}
	if showVersion {
		fmt.Println(version)
		return
	}

	if outFile != "" && !decryptOnly && !doEncrypt {
		fmt.Fprintln(os.Stderr, "Error: -o can only be used with --decrypt or --encrypt")
		os.Exit(1)
	}
	if exportQRDir != "" && decryptOnly {
		fmt.Fprintln(os.Stderr, "Error: --export-qr cannot be combined with --decrypt")
		os.Exit(1)
	}
	if showQR && decryptOnly {
		fmt.Fprintln(os.Stderr, "Error: --show-qr cannot be combined with --decrypt")
		os.Exit(1)
	}
	if showQR && exportQRDir != "" {
		fmt.Fprintln(os.Stderr, "Error: --show-qr cannot be combined with --export-qr")
		os.Exit(1)
	}
	if demoQR && (decryptOnly || doEncrypt || outFile != "" || exportQRDir != "" || showQR) {
		fmt.Fprintln(os.Stderr, "Error: --demo-qr cannot be combined with decrypt, encrypt, or other QR options")
		os.Exit(1)
	}
	if doEncrypt && (decryptOnly || exportQRDir != "" || showQR) {
		fmt.Fprintln(os.Stderr, "Error: --encrypt cannot be combined with --decrypt, --export-qr, or --show-qr")
		os.Exit(1)
	}

	args := fs.Args()
	if doEncrypt {
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "Error: --encrypt requires exactly one input file argument")
			os.Exit(1)
		}
		password := os.Getenv(passwordEnv)
		if password == "" {
			fmt.Fprintf(os.Stderr, "Error: %s environment variable is not set\n", passwordEnv)
			os.Exit(1)
		}
		plaintext, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input file:", err)
			os.Exit(1)
		}
		encrypted, err := encryptToExport(plaintext, password)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error encrypting:", err)
			os.Exit(1)
		}
		if outFile != "" {
			if err := os.WriteFile(outFile, encrypted, 0o600); err != nil {
				fmt.Fprintln(os.Stderr, "Error writing output file:", err)
				os.Exit(1)
			}
			fmt.Printf("Encrypted file written to %s\n", outFile)
		} else {
			os.Stdout.Write(encrypted)
		}
		return
	}
	if demoQR {
		if len(args) != 0 {
			fmt.Fprintln(os.Stderr, "Error: --demo-qr does not accept an export file or filter")
			os.Exit(1)
		}
		e, err := demoEntry()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		if err := showQRCodes([]entry{e}, ""); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	password := os.Getenv(passwordEnv)
	if password == "" {
		fmt.Fprintf(os.Stderr, "Error: %s environment variable is not set\n", passwordEnv)
		os.Exit(1)
	}

	exportFile := args[0]
	filter := strings.Join(args[1:], " ")

	plaintext, err := decryptExport(exportFile, password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error decrypting export:", err)
		os.Exit(1)
	}

	if decryptOnly {
		if outFile != "" {
			if err := os.WriteFile(outFile, plaintext, 0o600); err != nil {
				fmt.Fprintln(os.Stderr, "Error writing output file:", err)
				os.Exit(1)
			}
			fmt.Printf("Decrypted data written to %s\n", outFile)
		} else {
			os.Stdout.Write(plaintext)
		}
		return
	}

	entries, err := loadEntries(plaintext)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing entries:", err)
		os.Exit(1)
	}

	if exportQRDir != "" {
		written, err := exportQRCodes(entries, exportQRDir, filter)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error exporting QR codes:", err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %d QR code PNG(s) to %s\n", written, exportQRDir)
		fmt.Fprintln(os.Stderr, "WARNING: these PNG files contain your 2FA secrets in plaintext.")
		fmt.Fprintln(os.Stderr, "         Import them into your authenticator app(s), then delete")
		fmt.Fprintln(os.Stderr, "         them and empty your trash; do not leave them on disk.")
		return
	}

	if showQR {
		if err := showQRCodes(entries, filter); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	headers := []string{"ISSUER", "ACCOUNT", "CODE", "EXPIRES IN"}
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	underlines := make([]string, len(headers))
	for i, h := range headers {
		underlines[i] = strings.Repeat("=", len(h))
	}
	fmt.Fprintln(w, strings.Join(underlines, "\t"))
	matched := 0
	for _, e := range entries {
		if !matches(e, filter) {
			continue
		}
		code, remaining, err := generateCode(e)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error generating code for %s (%s): %v\n", e.issuer, e.account, err)
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%ds %s\n", e.issuer, e.account, code, remaining, time.Now().Format("15:04:05"))
		matched++
	}
	w.Flush()

	if matched == 0 {
		fmt.Println("No matching entries found.")
	}
}
