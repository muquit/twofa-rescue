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
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/poly1305"
)

// deriveArgonKey creates a 32-byte Argon2id key.
func deriveArgonKey(password, salt string, memLimit, opsLimit int) ([]byte, error) {
	if memLimit < 1024 || opsLimit < 1 {
		return nil, fmt.Errorf("invalid memory or operation limits")
	}

	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return nil, fmt.Errorf("invalid salt: %v", err)
	}

	key := argon2.IDKey([]byte(password), saltBytes, uint32(opsLimit), uint32(memLimit/1024), 1, 32)

	return key, nil
}

// Pure Go implementation of libsodium's
// crypto_secretstream_xchacha20poly1305. It follows the libsodium C source
// byte-for-byte and makes cross compilation possible without cgo.
// Secretstream has custom framing, so it uses the raw Poly1305 primitive.

const (
	secretStreamKeyBytes    = 32
	secretStreamHeaderBytes = 24
	secretStreamABytes      = 1 + 16 // framing tag byte + Poly1305 MAC
	secretStreamTagRekey    = 0x02
	secretStreamTagFinal    = 0x03 // TAG_PUSH | TAG_REKEY
)

var errSecretStreamAuth = errors.New("secretstream: message authentication failed (wrong password or corrupted data)")

// Same key and nonce state used by libsodium secretstream.
type secretStreamState struct {
	key   [32]byte
	nonce [12]byte
}

// Initialize the state for encryption or decryption.
func newSecretStreamState(header, key []byte) (*secretStreamState, error) {
	if len(header) != secretStreamHeaderBytes {
		return nil, fmt.Errorf("secretstream: invalid header length %d, want %d", len(header), secretStreamHeaderBytes)
	}
	if len(key) != secretStreamKeyBytes {
		return nil, fmt.Errorf("secretstream: invalid key length %d, want %d", len(key), secretStreamKeyBytes)
	}

	subKey, err := chacha20.HChaCha20(key, header[:16])
	if err != nil {
		return nil, fmt.Errorf("secretstream: deriving subkey: %w", err)
	}

	st := &secretStreamState{}
	copy(st.key[:], subKey)
	st.resetCounter()
	copy(st.nonce[4:], header[16:24])
	return st, nil
}

func (st *secretStreamState) resetCounter() {
	st.nonce[0], st.nonce[1], st.nonce[2], st.nonce[3] = 1, 0, 0, 0
}

func (st *secretStreamState) incrementCounter() {
	carry := uint16(1)
	for i := range 4 {
		carry += uint16(st.nonce[i])
		st.nonce[i] = byte(carry)
		carry >>= 8
	}
}

// Same operation as crypto_secretstream_xchacha20poly1305_rekey.
func (st *secretStreamState) rekey() error {
	var buf [40]byte // 32-byte key || 8-byte inonce
	copy(buf[:32], st.key[:])
	copy(buf[32:], st.nonce[4:12])

	cipher, err := chacha20.NewUnauthenticatedCipher(st.key[:], st.nonce[:])
	if err != nil {
		return err
	}
	cipher.XORKeyStream(buf[:], buf[:])

	copy(st.key[:], buf[:32])
	copy(st.nonce[4:12], buf[32:])
	st.resetCounter()
	return nil
}

// Authenticate and decrypt one secretstream chunk.
func (st *secretStreamState) pull(in, ad []byte) (plaintext []byte, tag byte, err error) {
	if len(in) < secretStreamABytes {
		return nil, 0, fmt.Errorf("secretstream: chunk too short (%d bytes)", len(in))
	}
	mlen := len(in) - secretStreamABytes

	// Counter 0 creates the one-time Poly1305 key.
	keyCipher, err := chacha20.NewUnauthenticatedCipher(st.key[:], st.nonce[:])
	if err != nil {
		return nil, 0, err
	}
	var block0 [64]byte
	keyCipher.XORKeyStream(block0[:], block0[:])
	var poly1305Key [32]byte
	copy(poly1305Key[:], block0[:32])
	mac := poly1305.New(&poly1305Key)

	mac.Write(ad)
	mac.Write(make([]byte, (16-len(ad)%16)%16))

	// Counter 1 recovers and authenticates the framing tag.
	frameCipher, err := chacha20.NewUnauthenticatedCipher(st.key[:], st.nonce[:])
	if err != nil {
		return nil, 0, err
	}
	frameCipher.SetCounter(1)
	var frame [64]byte
	frame[0] = in[0]
	frameCipher.XORKeyStream(frame[:], frame[:])
	tag = frame[0]
	frame[0] = in[0]
	mac.Write(frame[:])

	ciphertext := in[1 : 1+mlen]
	mac.Write(ciphertext)
	// libsodium uses mlen%16 here instead of normal 16-byte padding.
	// Keep this for byte-for-byte compatibility.
	mac.Write(make([]byte, mlen%16))

	var lenBuf [8]byte
	binary.LittleEndian.PutUint64(lenBuf[:], uint64(len(ad)))
	mac.Write(lenBuf[:])
	binary.LittleEndian.PutUint64(lenBuf[:], uint64(len(frame)+mlen))
	mac.Write(lenBuf[:])

	computedMAC := mac.Sum(nil)
	storedMAC := in[1+mlen:]
	if subtle.ConstantTimeCompare(computedMAC, storedMAC) != 1 {
		return nil, 0, errSecretStreamAuth
	}

	// Counter 2 and above decrypt the message.
	msgCipher, err := chacha20.NewUnauthenticatedCipher(st.key[:], st.nonce[:])
	if err != nil {
		return nil, 0, err
	}
	msgCipher.SetCounter(2)
	plaintext = make([]byte, mlen)
	msgCipher.XORKeyStream(plaintext, ciphertext)

	for i := range 8 {
		st.nonce[4+i] ^= computedMAC[i]
	}
	st.incrementCounter()

	needsRekey := tag&secretStreamTagRekey != 0
	if !needsRekey {
		needsRekey = st.nonce[0] == 0 && st.nonce[1] == 0 && st.nonce[2] == 0 && st.nonce[3] == 0
	}
	if needsRekey {
		if err := st.rekey(); err != nil {
			return nil, 0, err
		}
	}

	return plaintext, tag, nil
}

// Decrypt an Ente Auth export stored as one secretstream chunk.
func decryptChaCha20poly1305(data []byte, key []byte, header []byte) ([]byte, error) {
	state, err := newSecretStreamState(header, key)
	if err != nil {
		return nil, err
	}
	plaintext, _, err := state.pull(data, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// Encrypt and authenticate one secretstream chunk.
func (st *secretStreamState) push(m, ad []byte, tag byte) ([]byte, error) {
	mlen := len(m)

	// Counter 0 creates the one-time Poly1305 key.
	keyCipher, err := chacha20.NewUnauthenticatedCipher(st.key[:], st.nonce[:])
	if err != nil {
		return nil, err
	}
	var block0 [64]byte
	keyCipher.XORKeyStream(block0[:], block0[:])
	var poly1305Key [32]byte
	copy(poly1305Key[:], block0[:32])
	mac := poly1305.New(&poly1305Key)

	mac.Write(ad)
	mac.Write(make([]byte, (16-len(ad)%16)%16))

	// Counter 1 encrypts and authenticates the framing tag.
	frameCipher, err := chacha20.NewUnauthenticatedCipher(st.key[:], st.nonce[:])
	if err != nil {
		return nil, err
	}
	frameCipher.SetCounter(1)
	var frame [64]byte
	frame[0] = tag
	frameCipher.XORKeyStream(frame[:], frame[:])
	mac.Write(frame[:])

	out := make([]byte, secretStreamABytes+mlen)
	out[0] = frame[0]

	// Counter 2 and above encrypt the message.
	msgCipher, err := chacha20.NewUnauthenticatedCipher(st.key[:], st.nonce[:])
	if err != nil {
		return nil, err
	}
	msgCipher.SetCounter(2)
	ciphertext := out[1 : 1+mlen]
	msgCipher.XORKeyStream(ciphertext, m)

	mac.Write(ciphertext)
	// Same libsodium padding quirk as pull(); see the comment there.
	mac.Write(make([]byte, mlen%16))

	var lenBuf [8]byte
	binary.LittleEndian.PutUint64(lenBuf[:], uint64(len(ad)))
	mac.Write(lenBuf[:])
	binary.LittleEndian.PutUint64(lenBuf[:], uint64(len(frame)+mlen))
	mac.Write(lenBuf[:])

	computedMAC := mac.Sum(nil)
	copy(out[1+mlen:], computedMAC)

	for i := range 8 {
		st.nonce[4+i] ^= computedMAC[i]
	}
	st.incrementCounter()

	needsRekey := tag&secretStreamTagRekey != 0
	if !needsRekey {
		needsRekey = st.nonce[0] == 0 && st.nonce[1] == 0 && st.nonce[2] == 0 && st.nonce[3] == 0
	}
	if needsRekey {
		if err := st.rekey(); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// Encrypt data as one final secretstream chunk used by Ente Auth exports.
func encryptChaCha20poly1305(data []byte, key []byte, header []byte) ([]byte, error) {
	state, err := newSecretStreamState(header, key)
	if err != nil {
		return nil, err
	}
	return state.push(data, nil, secretStreamTagFinal)
}
