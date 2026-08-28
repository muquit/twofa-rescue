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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Decode the hardcoded test data.
func b64(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// Non-secret key containing bytes 0x00 through 0x1f.
var realLibsodiumKey = b64("AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=")

// Generated with the real libsodium C library using realLibsodiumKey.
// Unaligned lengths test the libsodium padding behavior.
var realLibsodiumVectors = []struct {
	name       string
	plaintext  []byte
	header     []byte
	ciphertext []byte
}{
	{
		name:       "empty",
		plaintext:  b64(""),
		header:     b64("XgyBnezigJOg8uRznCbMn+hv6NmQf0j4"),
		ciphertext: b64("MpgKR3OViWYefjEeIgdZZ9Y="),
	},
	{
		name:       "16_aligned",
		plaintext:  b64("QUFBQUFBQUFBQUFBQUFBQQ=="),
		header:     b64("z5QxYHvS9XcSZy0eh/57cH+HG+AVHJn7"),
		ciphertext: b64("euDkEvQgixtAc5psrVLo/kJIeVLTKcTee9DR2f414gTw"),
	},
	{
		name:       "32_aligned",
		plaintext:  b64("QUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUE="),
		header:     b64("meyZNwtU8VYBxarxPAEwu2luLZVPqHra"),
		ciphertext: b64("x6/tVAq5x3sfw0xT0sESmBPlS0OdYOZs/XEqYBFiASg4uBDtmOsIZWwZOfM4MQ2lTA=="),
	},
	{
		name:       "17_unaligned",
		plaintext:  b64("QUFBQUFBQUFBQUFBQUFBQUE="),
		header:     b64("/nTkTm0ssEcRxrQPiRcEC1ioB6xTNuDe"),
		ciphertext: b64("V/8FmmptBpfaGG0VcDtrE72n9dgehVsYp9tGyYDigppSQQ=="),
	},
	{
		name:       "otpauth_lines",
		plaintext:  b64("b3RwYXV0aDovL3RvdHAvR2l0SHViOmFsaWNlQGV4YW1wbGUuY29tP3NlY3JldD1KQlNXWTNEUEVIUEszUFhQJmlzc3Vlcj1HaXRIdWIKb3RwYXV0aDovL3RvdHAvQVdTOmFsaWNlQGV4YW1wbGUuY29tP3NlY3JldD1LUlNYRzVDVE1WUlhFWkxVJmlzc3Vlcj1BV1MK"),
		header:     b64("ENNoaHaiqHV0RLDd2SK9nrNNGrJfEa5e"),
		ciphertext: b64("uSD5Nyyt8x099Yf+ITqGHJleHFCImzUo613yTErz/bxxMZFKzVw89TXQv+yoHRAQ9K2CFQf/ENj4u6ieTVHhmSE3LtTpCKDQU463b/ZjWK6ZaXfwcxvFq5IeIClxBxkulY4kzTiY7LRkLyxNwhItqDc2VVE2+aEJGEEoUtIzXxvzwB82XVClq9MlrlGwiG0EietTLe//4p8XnxbEAR457bXa+eIpjVE="),
	},
	{
		// Spans several ChaCha20 blocks in one call.
		name:       "large_1400_unaligned",
		plaintext:  bytes.Repeat([]byte("otpauth://totp/x?secret=ABC\n"), 50), // 1400 bytes
		header:     b64("7xkayn5uNinxyj441GQHiK6KeTHG7aAE"),
		ciphertext: b64("tMqRslXVAY//vgARFXUj3o5nx55vY5nHkFhciNBbnH2Jmo6MKReRiGdl7Yxl/nCHEDngW9vW0vLhAD4xNuRcAyfI7yU2fBACqLX/mikEnvBpTzzJ1tgKXM3tTNEbQWiD2dss3vJur8vvZaei0oGXrMJr145hwWy8XEw5dg9FgRykk+8XGNxp8p4EP0JcKm5Ke8+uht+7U+SqdOhQ8NkNCoZM7OpVCnbOhthnVXtoZ1AfVpzBDaYupD5hXQF0iZKNjk484MlDt+UU2VZZQLok/Klc71Fk0eCvZL2RLpTPYeWco3nKWHgQniVM8vySdsBYg70CgdZIuMnC+VhaEVtpnW4o4kpRpLYSKKu6YEG5kwgS6e1aQMNjkUU/4rfOIL9Y6wQQHxJLiDT+iglDwKUqay+fXqm4+G/t7uvNEMfPhVMgFuAEA/u30diFhbpDkz9J5kWYZjOpr4r9f8h/vkbfqptSsy2TXPxm8mALlpjfHbKruwxONDY5ZZwX53GjShK5TsETPlVfOJC+C1cQbYYy8dmvWm5OKry5fl/1MsvoE/+8NIZk55YQ/PnZ5qQgKix2gheX7k/5/+HzItAlw5QHQADp1z/xFvJAU1T44+rjBe+EXlODnh18K8Zds0CCG/T9iTiI5NyL9ZAo4TyFwn5sUmQ4ZW7JCJgKYfehwc1p3DOR8KWmmw5ZSkDr7+d50UwZfvYzBxC7mL/KbysahYYQ0PXHysDSdxWLtdDpcAZF7XeScfAU3yiYFaSK5Opbv5Teghev57SBnRZE4jLQmVe2zEsoNqrXIsn1RtbRf0rQD/eJFK6WbfoGaI+b0RbQEW6F6HQtlBhHZ/thoygAuFs4+fG1loRf+BQqFB0oY8G5ClcC1DVedboF2IWrZ6JmsOCGwxjdDO+NezyUjCA27xKr9vvIk2A6w8x9yrq2sqdnQs1qgQ9dkfuzWtnBtE5hO/QBH0Z23+5kaOeth0WGeh2paJJeulXSZnJcG8K1QXUwVMWl4iycCHx2rAX4jTYlEEiaGBwoPRpQfI0ybfpXzVi0FFO89C4MZOUhPzYTJmAcPcLFw9rThHNopNXRSwXam1yT/Yd+pjgjoheNfyCFkqnzm35CwUN/9t+R5ooG9IXjCQTe1ARSIir6U7wYQUTB9O+5DVsMDKOQ6GT7tS6nEPqwGR4r+8q7C668h1yfSvpMO0imhHIfV6Q3MdRfPIlPsfFTjGFBST7B+LuboyUDvGWDEo5mCOFb17bGphNDVwLCK26NdbVudqTxdrb6mEzs6vfKgFLnM4ZlVu8m1bIc3Acrx3Yqa+aiuZjXHASl5lPFq/4NYgIu9u+bsj/p8PwfrXEUKD68+lWHuzMN7Nx0WzsKT6pysZ2Pc1+PTfbE868Y4it+EdA2lfVpbKS5BdR9gmmZTKLgTlOXJl4G6CBhJY18GFCxL3bGaLQH7KCWLyQ0nzr3m7ncpdBconWOn1maHIeigTiPmVt1Jm5XuamADrmS554LEhHNf4nNgVoKueoSTB1neSdjGX4VIujpSY8jKVyTxwkhdJNOazi4OCC8glzU8+VJo9iPeaF03JxOxJe1gx37RvFOPo3dwPRI/v9JXbqV+Dsv4Op5Ou3XJXT8MhPOdcOMWsqCqKPBo4059QI36ZuxoVqh6zIUo5YjvuYpYzmao5o3UgWy6ZUv53mlH4Ink9axL1XSCRqv+zY0wDu+WPXIQV7plGetnqvEbaNgZeo9oBpfaafs1X/ji5dwesa5vgk+QuBXYic+ccq3aa2vXJgBXTPMUWvtXS+52rPXLj7TvtB+BKc1xoa4Il6f1d/3jcJ9s1wZ4vDnCOh5e6kZ9NisJxmaUertvUcAprNb3wtladJUSfaALkz2lwefTJle0BNDsf/oI9Youw=="),
	},
}

// Decrypt test data created by the real libsodium C library.
func TestPullMatchesRealLibsodium(t *testing.T) {
	for _, v := range realLibsodiumVectors {
		t.Run(v.name, func(t *testing.T) {
			got, err := decryptChaCha20poly1305(v.ciphertext, realLibsodiumKey, v.header)
			if err != nil {
				t.Fatalf("decryptChaCha20poly1305: %v", err)
			}
			if !bytes.Equal(got, v.plaintext) {
				t.Fatalf("plaintext mismatch:\n got:  %q\n want: %q", got, v.plaintext)
			}
		})
	}
}

// Compare encrypted output byte-for-byte with real libsodium output.
func TestPushMatchesRealLibsodium(t *testing.T) {
	for _, v := range realLibsodiumVectors {
		t.Run(v.name, func(t *testing.T) {
			got, err := encryptChaCha20poly1305(v.plaintext, realLibsodiumKey, v.header)
			if err != nil {
				t.Fatalf("encryptChaCha20poly1305: %v", err)
			}
			if !bytes.Equal(got, v.ciphertext) {
				t.Fatalf("ciphertext mismatch:\n got:  %x\n want: %x", got, v.ciphertext)
			}
		})
	}
}

// Test encryption and decryption with different message sizes.
func TestRoundTrip(t *testing.T) {
	sizes := []int{0, 1, 15, 16, 17, 63, 64, 65, 1000, 8192}
	for _, size := range sizes {
		key := make([]byte, secretStreamKeyBytes)
		header := make([]byte, secretStreamHeaderBytes)
		plaintext := make([]byte, size)
		for _, b := range [][]byte{key, header, plaintext} {
			if _, err := rand.Read(b); err != nil {
				t.Fatalf("rand.Read: %v", err)
			}
		}

		ciphertext, err := encryptChaCha20poly1305(plaintext, key, header)
		if err != nil {
			t.Fatalf("size %d: encrypt: %v", size, err)
		}
		if len(ciphertext) != len(plaintext)+secretStreamABytes {
			t.Fatalf("size %d: ciphertext length = %d, want %d", size, len(ciphertext), len(plaintext)+secretStreamABytes)
		}

		got, err := decryptChaCha20poly1305(ciphertext, key, header)
		if err != nil {
			t.Fatalf("size %d: decrypt: %v", size, err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Fatalf("size %d: round-trip mismatch", size)
		}
	}
}

// A wrong key must fail authentication.
func TestWrongKeyFailsAuth(t *testing.T) {
	v := realLibsodiumVectors[len(realLibsodiumVectors)-1] // non-empty vector
	wrongKey := make([]byte, secretStreamKeyBytes)
	copy(wrongKey, realLibsodiumKey)
	wrongKey[0] ^= 0xff

	if _, err := decryptChaCha20poly1305(v.ciphertext, wrongKey, v.header); err == nil {
		t.Fatal("expected an authentication error with the wrong key, got nil")
	}
}

// Corrupted ciphertext must fail authentication.
func TestCorruptedCiphertextFailsAuth(t *testing.T) {
	v := realLibsodiumVectors[len(realLibsodiumVectors)-1]
	corrupted := append([]byte(nil), v.ciphertext...)
	corrupted[len(corrupted)/2] ^= 0xff

	if _, err := decryptChaCha20poly1305(corrupted, realLibsodiumKey, v.header); err == nil {
		t.Fatal("expected an authentication error for corrupted ciphertext, got nil")
	}
}

// Test the complete export pipeline with Ente Auth's Argon2id parameters.
// This test is slow and skipped with go test -short.
func TestEncryptToExportRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow Argon2id round trip in -short mode")
	}

	plaintext := []byte("otpauth://totp/GitHub:alice@example.com?secret=JBSWY3DPEHPK3PXP&issuer=GitHub\n")
	password := "correct horse battery staple"

	encrypted, err := encryptToExport(plaintext, password)
	if err != nil {
		t.Fatalf("encryptToExport: %v", err)
	}

	var export Export
	if err := json.Unmarshal(encrypted, &export); err != nil {
		t.Fatalf("encryptToExport did not produce valid JSON: %v", err)
	}
	if export.Version != 1 {
		t.Fatalf("version = %d, want 1", export.Version)
	}
	if export.KDFParams.MemLimit != encryptMemLimitBytes || export.KDFParams.OpsLimit != encryptOpsLimit {
		t.Fatalf("kdfParams = %+v, want Ente Auth's own defaults (memLimit=%d, opsLimit=%d)",
			export.KDFParams, encryptMemLimitBytes, encryptOpsLimit)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "export.json")
	if err := os.WriteFile(path, encrypted, 0o600); err != nil {
		t.Fatalf("writing temp export file: %v", err)
	}

	got, err := decryptExport(path, password)
	if err != nil {
		t.Fatalf("decryptExport: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round-trip mismatch:\n got:  %q\n want: %q", got, plaintext)
	}

	if _, err := decryptExport(path, "wrong password"); err == nil {
		t.Fatal("expected decryptExport to fail with the wrong password, got nil")
	}
}
