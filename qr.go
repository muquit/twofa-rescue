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
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

const qrImageSize = 300

// provisioningURL creates a TOTP URI accepted by Google Authenticator.
// Google requires no Base32 padding and the issuer in the label and query.
func provisioningURL(e entry) string {
	secret := strings.ToUpper(e.key.Secret())
	secret = strings.Map(func(r rune) rune {
		if r == '=' || r == ' ' || r == '\t' || r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, secret)

	label := e.account
	if e.issuer != "" {
		label = e.issuer + ":" + e.account
	}
	query := url.Values{"secret": {secret}}
	if e.issuer != "" {
		query.Set("issuer", e.issuer)
	}
	if algorithm := e.key.Algorithm().String(); algorithm != "SHA1" {
		query.Set("algorithm", algorithm)
	}
	if digits := e.key.Digits().Length(); digits != 6 {
		query.Set("digits", fmt.Sprint(digits))
	}
	if period := e.key.Period(); period != 0 && period != 30 {
		query.Set("period", fmt.Sprint(period))
	}

	return "otpauth://totp/" + url.PathEscape(label) + "?" + query.Encode()
}

// Quartile correction helps when a QR code is scanned from a screen.
const qrErrorCorrection = qr.Q

// Use a smaller QR code in Apple Terminal because each module needs one row.
const appleTerminalQRErrorCorrection = qr.L

// generateQRCode creates and scales a QR code image.
func generateQRCode(otpauthURL string, width, height int) (image.Image, error) {
	code, err := qr.Encode(otpauthURL, qrErrorCorrection, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("encoding QR code: %w", err)
	}
	return barcode.Scale(code, width, height)
}

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFilename(s string) string {
	s = unsafeFilenameChars.ReplaceAllString(strings.TrimSpace(s), "_")
	s = strings.Trim(s, "_")
	if s == "" {
		s = "unnamed"
	}
	return s
}

// exportQRCodes writes one PNG file for each matching entry.
func exportQRCodes(entries []entry, dir, filter string) (int, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, fmt.Errorf("creating output directory: %w", err)
	}

	seen := make(map[string]int)
	written := 0
	for _, e := range entries {
		if !matches(e, filter) {
			continue
		}

		img, err := generateQRCode(provisioningURL(e), qrImageSize, qrImageSize)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping QR code for %s (%s): %v\n", e.issuer, e.account, err)
			continue
		}

		name := sanitizeFilename(e.issuer)
		if e.account != "" {
			name += "-" + sanitizeFilename(e.account)
		}
		seen[name]++
		if n := seen[name]; n > 1 {
			name = fmt.Sprintf("%s-%d", name, n)
		}
		path := filepath.Join(dir, name+".png")

		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			return written, fmt.Errorf("creating %s: %w", path, err)
		}
		err = png.Encode(f, img)
		closeErr := f.Close()
		if err != nil {
			return written, fmt.Errorf("writing %s: %w", path, err)
		}
		if closeErr != nil {
			return written, fmt.Errorf("closing %s: %w", path, closeErr)
		}
		written++
	}
	return written, nil
}

// QR specification requires a four-module white border.
const qrQuietZone = 4

const (
	ansiForceBlackOnWhite  = "\x1b[30;47m"
	ansiReset              = "\x1b[0m"
	itermImagePrefix       = "\x1b]1337;File=inline=1;preserveAspectRatio=1:"
	itermImageSuffix       = "\a\n"
	terminalQRModulePixels = 8
	ansiBlockDark          = "\x1b[40m"
	ansiBlockLight         = "\x1b[47m"
)

// Resize Apple Terminal before displaying any text or the QR code.
// Resizing after printing can scroll text off the screen.
func ensureAppleTerminalSize(otpauthURL string) error {
	if os.Getenv("TERM_PROGRAM") != "Apple_Terminal" {
		return nil
	}
	code, err := qr.Encode(otpauthURL, appleTerminalQRErrorCorrection, qr.Auto)
	if err != nil {
		return fmt.Errorf("encoding QR code: %w", err)
	}
	return ensureTerminalFits(code)
}

// Use the best QR renderer available for the current terminal.
func renderQRTerminal(w io.Writer, otpauthURL string) error {
	termProgram := os.Getenv("TERM_PROGRAM")
	errorCorrection := qrErrorCorrection
	if termProgram == "Apple_Terminal" {
		errorCorrection = appleTerminalQRErrorCorrection
	}
	code, err := qr.Encode(otpauthURL, errorCorrection, qr.Auto)
	if err != nil {
		return fmt.Errorf("encoding QR code: %w", err)
	}
	switch termProgram {
	case "iTerm.app":
		if debugMode {
			fmt.Fprintf(os.Stderr, "[debug] TERM_PROGRAM=%q TERM=%q: using iTerm2 inline image protocol\n", termProgram, os.Getenv("TERM"))
		}
		return renderQRIterm(w, code)
	case "Apple_Terminal":
		if debugMode {
			fmt.Fprintf(os.Stderr, "[debug] TERM_PROGRAM=%q TERM=%q: using Apple Terminal solid-block fallback\n", termProgram, os.Getenv("TERM"))
		}
		return renderQRAppleTerminal(w, code)
	default:
		if debugMode {
			fmt.Fprintf(os.Stderr, "[debug] TERM_PROGRAM=%q TERM=%q: no known image protocol, using Unicode block fallback\n", termProgram, os.Getenv("TERM"))
		}
		return renderQRBlocks(w, code)
	}
}

func renderQRIterm(w io.Writer, code barcode.Barcode) error {
	bounds := code.Bounds()
	modules := bounds.Dx() + 2*qrQuietZone
	size := modules * terminalQRModulePixels
	img := image.NewGray(image.Rect(0, 0, size, size))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, _, _, _ := code.At(x, y).RGBA()
			if r != 0 {
				continue
			}
			px := (x - bounds.Min.X + qrQuietZone) * terminalQRModulePixels
			py := (y - bounds.Min.Y + qrQuietZone) * terminalQRModulePixels
			for dy := range terminalQRModulePixels {
				for dx := range terminalQRModulePixels {
					img.SetGray(px+dx, py+dy, color.Gray{Y: 0})
				}
			}
		}
	}

	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		return fmt.Errorf("encoding terminal QR image: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(pngData.Bytes())
	_, err := fmt.Fprint(w, itermImagePrefix, encoded, itermImageSuffix)
	return err
}

func renderQRBlocks(w io.Writer, code barcode.Barcode) error {
	bounds := code.Bounds()
	isDark := func(x, y int) bool {
		if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
			return false // quiet zone is always light
		}
		r, _, _, _ := code.At(x, y).RGBA()
		return r == 0 // color.Black
	}

	minX, maxX := bounds.Min.X-qrQuietZone, bounds.Max.X+qrQuietZone
	minY, maxY := bounds.Min.Y-qrQuietZone, bounds.Max.Y+qrQuietZone
	fmt.Fprint(w, ansiForceBlackOnWhite)
	for y := minY; y < maxY; y += 2 {
		for x := minX; x < maxX; x++ {
			top, bottom := isDark(x, y), isDark(x, y+1)
			switch {
			case top && bottom:
				fmt.Fprint(w, "█")
			case top && !bottom:
				fmt.Fprint(w, "▀")
			case !top && bottom:
				fmt.Fprint(w, "▄")
			default:
				fmt.Fprint(w, " ")
			}
		}
		fmt.Fprintln(w)
	}
	fmt.Fprint(w, ansiReset)
	return nil
}

// Apple Terminal displays one prompt line below the QR code.
const appleTerminalExtraUILines = 1

// Apple Terminal takes a little time to complete a resize request.
const (
	ensureTerminalFitsPollInterval = 100 * time.Millisecond
	ensureTerminalFitsTimeout      = 2 * time.Second
)

// ensureTerminalFits resizes Apple Terminal if the QR code will not fit.
func ensureTerminalFits(code barcode.Barcode) error {
	bounds := code.Bounds()
	neededRows := bounds.Dy() + 2*qrQuietZone + appleTerminalExtraUILines
	neededCols := 2 * (bounds.Dx() + 2*qrQuietZone)
	rows, cols, ok := terminalSize(int(os.Stdout.Fd()))
	if !ok {
		return fmt.Errorf("could not determine Apple Terminal window size")
	}
	if rows >= neededRows && cols >= neededCols {
		return nil
	}
	fmt.Fprintf(os.Stdout, "\x1b[8;%d;%dt", neededRows, neededCols)
	if debugMode {
		fmt.Fprintf(os.Stderr, "[debug] requested terminal resize to %dx%d (cols x rows), was %dx%d; polling for up to %v\n", neededCols, neededRows, cols, rows, ensureTerminalFitsTimeout)
	}
	deadline := time.Now().Add(ensureTerminalFitsTimeout)
	for {
		time.Sleep(ensureTerminalFitsPollInterval)
		rows, cols, ok = terminalSize(int(os.Stdout.Fd()))
		if debugMode {
			fmt.Fprintf(os.Stderr, "[debug] terminal size now %dx%d (cols x rows)\n", cols, rows)
		}
		if ok && rows >= neededRows && cols >= neededCols {
			return nil
		}
		if time.Now().After(deadline) {
			break
		}
	}

	if !ok || rows < neededRows || cols < neededCols {
		return fmt.Errorf("Terminal too small: Got %dx%d, Need %dx%d\n"+
			"Try a smaller font (Cmd-). Apple Terminal limitation, see:\n"+
			"https://github.com/muquit/twofa-rescue#faq",
			cols, rows, neededCols, neededRows)
	}
	/*
		if !ok || rows < neededRows || cols < neededCols {
			return fmt.Errorf("Attempted to resize Terminal to %dx%d, but it could only reach %dx%d.\n"+
				"This QR code needs at least %dx%d. Try a smaller Terminal font (Cmd+-).\n"+
				"This is a limitation of Apple Terminal. See FAQ for details:\n"+
				"https://github.com/muquit/twofa-rescue#faq", neededCols, neededRows, cols, rows, neededCols, neededRows)
		}
	*/
	return nil
}

// Apple Terminal distorts Unicode block characters. Use two-character-wide
// ANSI background colors to keep the QR modules square.
func renderQRAppleTerminal(w io.Writer, code barcode.Barcode) error {
	bounds := code.Bounds()
	isDark := func(x, y int) bool {
		if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
			return false // quiet zone is always light
		}
		r, _, _, _ := code.At(x, y).RGBA()
		return r == 0 // color.Black
	}

	minX, maxX := bounds.Min.X-qrQuietZone, bounds.Max.X+qrQuietZone
	minY, maxY := bounds.Min.Y-qrQuietZone, bounds.Max.Y+qrQuietZone
	for y := minY; y < maxY; y++ {
		currentDark := false
		fmt.Fprint(w, ansiBlockLight)
		for x := minX; x < maxX; x++ {
			dark := isDark(x, y)
			if dark != currentDark {
				if dark {
					fmt.Fprint(w, ansiBlockDark)
				} else {
					fmt.Fprint(w, ansiBlockLight)
				}
				currentDark = dark
			}
			fmt.Fprint(w, "  ")
		}
		fmt.Fprintln(w, ansiReset)
	}
	return nil
}

// Return true if the file is an interactive terminal.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

const (
	ansiEnterAltScreen = "\x1b[?1049h"
	ansiExitAltScreen  = "\x1b[?1049l"
	ansiHomeAndClear   = "\x1b[H\x1b[2J"
)

// Display matching QR codes one at a time. Enter advances and Ctrl+C quits.
// Interactive terminals use the alternate screen to keep QR codes out of
// the scrollback history.
func showQRCodes(entries []entry, filter string) error {
	var matched []entry
	for _, e := range entries {
		if matches(e, filter) {
			matched = append(matched, e)
		}
	}
	if len(matched) == 0 {
		return fmt.Errorf("no matching entries")
	}

	interactive := isTerminal(os.Stdout)

	// Save the original size in case Apple Terminal is enlarged.
	var startRows, startCols int
	haveStartSize := false
	if os.Getenv("TERM_PROGRAM") == "Apple_Terminal" {
		startRows, startCols, haveStartSize = terminalSize(int(os.Stdout.Fd()))
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if interactive {
				fmt.Print(ansiReset)
			}
			if haveStartSize {
				if rows, cols, ok := terminalSize(int(os.Stdout.Fd())); ok && (rows > startRows || cols > startCols) {
					fmt.Fprintf(os.Stdout, "\x1b[8;%d;%dt", startRows, startCols)
				}
			}
			if interactive {
				fmt.Print(ansiExitAltScreen)
			}
		})
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	done := make(chan struct{})
	go func() {
		select {
		case <-sigCh:
			cleanup()
			os.Exit(130)
		case <-done:
		}
	}()
	defer func() {
		close(done)
		signal.Stop(sigCh)
		cleanup()
	}()

	if interactive {
		fmt.Print(ansiEnterAltScreen)
	}

	stdin := bufio.NewReader(os.Stdin)
	for i, e := range matched {
		otpauthURL := provisioningURL(e)
		appleTerminal := os.Getenv("TERM_PROGRAM") == "Apple_Terminal"
		if interactive {
			if err := ensureAppleTerminalSize(otpauthURL); err != nil {
				return err
			}
		}
		if interactive {
			fmt.Print(ansiHomeAndClear)
		}
		if !appleTerminal {
			fmt.Printf("%s (%s)\n\n", e.issuer, e.account)
		}
		if err := renderQRTerminal(os.Stdout, otpauthURL); err != nil {
			return err
		}
		if code, remaining, err := generateCode(e); err == nil && !appleTerminal {
			fmt.Printf("\nCurrent code: %s (expires in %ds)\n", code, remaining)
		}

		identity := ""
		if appleTerminal {
			identity = fmt.Sprintf(" %s (%s) -", e.issuer, e.account)
		}
		promptPrefix := "\n"
		if appleTerminal {
			promptPrefix = ""
		}
		if i < len(matched)-1 {
			fmt.Printf("%s[%d/%d]%s Press Enter for next QR code (Ctrl+C to quit)...", promptPrefix, i+1, len(matched), identity)
		} else {
			fmt.Printf("%s[%d/%d]%s Press Enter to finish...", promptPrefix, i+1, len(matched), identity)
		}
		if _, err := stdin.ReadString('\n'); err != nil {
			return nil
		}
	}
	return nil
}
