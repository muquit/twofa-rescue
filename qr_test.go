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
	"bytes"
	"encoding/base64"
	"image/png"
	"net/url"
	"strings"
	"testing"

	"github.com/boombuler/barcode/qr"
	"github.com/pquerna/otp"
)

const testOTPURI = "otpauth://totp/Example:alice@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Example"

func TestProvisioningURLCanonicalizesForGoogleAuthenticator(t *testing.T) {
	key, err := otp.NewKeyFromURL("otpauth://totp/alice@example.com?secret=jbsw%20y3dp%20ehpk3pxp%3D&issuer=Example")
	if err != nil {
		t.Fatal(err)
	}
	got := provisioningURL(entry{issuer: key.Issuer(), account: key.AccountName(), key: key})
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.EscapedPath() != "/Example:alice@example.com" {
		t.Fatalf("label = %q", u.EscapedPath())
	}
	if secret := u.Query().Get("secret"); secret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("secret was not canonicalized: %q", secret)
	}
	if issuer := u.Query().Get("issuer"); issuer != "Example" {
		t.Fatalf("issuer = %q", issuer)
	}
}

func TestDemoEntryIsPublicCanonicalTestAccount(t *testing.T) {
	e, err := demoEntry()
	if err != nil {
		t.Fatal(err)
	}
	if e.issuer != "twofa-rescue Demo" || e.account != "demo@example.com" {
		t.Fatalf("unexpected demo identity: %q %q", e.issuer, e.account)
	}
	const canonical = "otpauth://totp/twofa-rescue%20Demo:demo@example.com?issuer=twofa-rescue+Demo&secret=JBSWY3DPEHPK3PXP"
	if got := provisioningURL(e); got != canonical {
		t.Fatalf("demo provisioning URL = %q", got)
	}
}

func TestRenderQRTerminalUsesItermInlineImage(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	code, err := qr.Encode(testOTPURI, qrErrorCorrection, qr.Auto)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := renderQRTerminal(&out, testOTPURI); err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	if !strings.HasPrefix(rendered, itermImagePrefix) || !strings.HasSuffix(rendered, itermImageSuffix) {
		t.Fatal("output is not an iTerm2 inline image")
	}

	b64 := strings.TrimSuffix(strings.TrimPrefix(rendered, itermImagePrefix), itermImageSuffix)
	pngBytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("decoding inline image: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decoding PNG: %v", err)
	}
	wantSize := (code.Bounds().Dx() + 2*qrQuietZone) * terminalQRModulePixels
	if img.Bounds().Dx() != wantSize || img.Bounds().Dy() != wantSize {
		t.Fatalf("PNG is %v, want %dx%d", img.Bounds(), wantSize, wantSize)
	}
	if gray, _, _, _ := img.At(0, 0).RGBA(); gray != 0xffff {
		t.Fatal("PNG quiet zone is not white")
	}
}

func TestRenderQRTerminalFallsBackToBlocks(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	var out bytes.Buffer
	if err := renderQRTerminal(&out, testOTPURI); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), ansiForceBlackOnWhite) || !strings.ContainsAny(out.String(), "▀▄█") {
		t.Fatal("non-iTerm, non-Apple-Terminal output did not use the block fallback")
	}
}

func TestRenderQRTerminalUsesAppleTerminalSolidBlocks(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "Apple_Terminal")
	code, err := qr.Encode(testOTPURI, appleTerminalQRErrorCorrection, qr.Auto)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := renderQRTerminal(&out, testOTPURI); err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, ansiBlockDark) && !strings.Contains(rendered, ansiBlockLight) {
		t.Fatal("Apple Terminal output did not use the solid-block renderer")
	}
	if strings.ContainsAny(rendered, "▀▄█") {
		t.Fatal("Apple Terminal output should not use ▀▄█ glyphs")
	}
	wantRows := code.Bounds().Dy() + 2*qrQuietZone
	if got := strings.Count(rendered, "\n"); got != wantRows {
		t.Fatalf("Apple Terminal output has %d rows, want %d", got, wantRows)
	}
	if got := strings.Count(rendered, ansiReset); got != wantRows {
		t.Fatalf("Apple Terminal output has %d ANSI resets, want one per row (%d)", got, wantRows)
	}
}

func TestAppleTerminalErrorCorrectionReducesSymbolSize(t *testing.T) {
	compact, err := qr.Encode(testOTPURI, appleTerminalQRErrorCorrection, qr.Auto)
	if err != nil {
		t.Fatal(err)
	}
	standard, err := qr.Encode(testOTPURI, qrErrorCorrection, qr.Auto)
	if err != nil {
		t.Fatal(err)
	}
	if compact.Bounds().Dx() >= standard.Bounds().Dx() {
		t.Fatalf("Apple Terminal QR is %d modules wide, standard QR is %d", compact.Bounds().Dx(), standard.Bounds().Dx())
	}
}
