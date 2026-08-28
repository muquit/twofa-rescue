# Ente Auth Encrypted Export: Algorithm and How We Decrypted It - original Claude Code notes

## Export file format

The encrypted export (`ente-auth-codes-*.txt`) is a JSON document with four fields:

```json
{
  "version": 1,
  "kdfParams": {
    "memLimit": 1073741824,
    "opsLimit": 4,
    "salt": "base64..."
  },
  "encryptedData": "base64...",
  "encryptionNonce": "base64..."
}
```

- `version`: export format version (1 is the only version handled here).
- `kdfParams`: parameters used to derive the encryption key from the user's export password.
- `encryptedData`: base64-encoded ciphertext.
- `encryptionNonce`: base64-encoded stream header (see below; not a plain AEAD nonce).

This matches Ente's own documented format
([`encrypted_export.md`](https://github.com/ente-io/auth/blob/main/migration-guides/encrypted_export.md))
and their reference decrypt tool
([`migration-guides/decrypt`](https://github.com/ente-io/auth/tree/main/migration-guides/decrypt)).

## Key derivation: Argon2id

The export password is never used directly as the encryption key. Instead:

```
key = Argon2id(password, salt, opsLimit, memLimit/1024 KiB, parallelism=1, keyLen=32)
```

- Algorithm: **Argon2id**
- `salt`: random, base64-decoded from `kdfParams.salt`
- `opsLimit`: time cost (iterations), from `kdfParams.opsLimit`
- `memLimit`: memory cost in **bytes** in the JSON, converted to KiB (`memLimit/1024`) for
  the Go `argon2.IDKey` call, which expects memory in KiB
- Output: 32-byte key (matches libsodium's `crypto_secretstream_xchacha20poly1305` key size)

We used Go's standard `golang.org/x/crypto/argon2` package for this step (pure Go, no cgo
needed), and it implements Argon2id per RFC 9106 identically to libsodium's `crypto_pwhash`
with the Argon2id algorithm ID.

## Encryption: libsodium `crypto_secretstream_xchacha20poly1305`

The ciphertext isn't encrypted with plain XChaCha20-Poly1305 AEAD; it uses libsodium's
**secretstream** construction, which is a chunked/streaming variant built on
XChaCha20-Poly1305:

- Each chunk is encrypted and authenticated (Poly1305 tag) individually.
- A 24-byte **stream header** (not a nonce in the traditional AEAD sense) is generated once
  per stream and is what's stored, base64-encoded, in `encryptionNonce`.
- The header + derived key together initialize the stream state used to decrypt (and
  verify) every chunk.
- The last chunk carries a "final" tag so the decoder knows the stream is complete.
- Ente writes the entire export as a single chunk (tag `FINAL`), so decrypting is one
  `pull()` call, not a multi-chunk loop.

This is why the field is called `encryptionNonce` in the JSON but is actually a
secretstream header, a naming detail that only matters if you're reimplementing this
without a library that already understands the construction.

### The construction, in detail (needed because we now reimplement it)

Per chunk, `crypto_secretstream_xchacha20poly1305_pull` does the following, given the
stream state's 32-byte subkey `k` and 12-byte nonce (a 4-byte little-endian block counter
followed by an 8-byte "inonce"):

1. Generate a 64-byte ChaCha20 keystream block at counter 0 (`ChaCha20(k, nonce, ic=0)`).
   Its first 32 bytes are the one-time Poly1305 key for this chunk.
2. Generate a second 64-byte keystream block at counter 1. XOR it with a 64-byte buffer
   containing the chunk's leading **framing byte** (a tag: `MESSAGE`=0x00, `PUSH`=0x01,
   `REKEY`=0x02, `FINAL`=0x03) followed by 63 zero bytes; this recovers the plaintext tag.
   The *raw ciphertext* framing byte (not the recovered tag) plus the rest of that 64-byte
   block are what get authenticated.
3. Feed Poly1305: associated data (padded to a 16-byte boundary) + the 64-byte framing
   block from step 2 + the raw ciphertext body + a padding gap + two 8-byte
   little-endian length fields (AD length, then `64 + ciphertext length`).
4. Compare the computed MAC against the trailing 16 bytes of the chunk. Mismatch = wrong
   password/corrupted data, full stop; no plaintext is released.
5. Only once authenticated: decrypt the ciphertext body by XORing with keystream blocks
   starting at counter 2.
6. Advance the state: XOR the nonce's inonce half with the first 8 bytes of the MAC,
   increment the counter, and rekey (derive a fresh subkey + inonce, reset the counter) if
   the tag had the `REKEY` bit set or the counter wrapped to zero.

**The one part that's genuinely a quirk, not a design choice**: libsodium's own C source
pads the ciphertext body with `(0x10 - (sizeof block) + mlen) & 0xf` bytes, and has a
comment on that exact line admitting `/* should have been (0x10 - (sizeof block + mlen)) &
0xf to keep input blocks aligned */`. Since `sizeof block` is 64 (a multiple of 16), the
formula actually used reduces to `mlen % 16`, not the "correct" pad-to-16-byte-boundary
length `(16 - mlen % 16) % 16` the comment says it should be. This is now part of the wire
format: any reimplementation has to replicate the quirky formula, not the sensible one, or
every message whose length isn't already a multiple of 16 fails to authenticate. We found
this the hard way; see Verification below.

## Decryption steps (what our tool does)

1. Parse the export JSON, check `version == 1`.
2. Base64-decode `kdfParams.salt`, `encryptedData`, `encryptionNonce`.
3. Derive the 32-byte key: `Argon2id(password, salt, opsLimit, memLimit/1024, 1, 32)`.
4. Derive the stream subkey via HChaCha20(key, header[0:16]) and seed the nonce's inonce
   half from header[16:24], per `init_pull` above.
5. Run `pull()` (as detailed above) on the ciphertext as a single chunk. The MAC check
   fails immediately with an authentication error on a wrong password; it never returns
   corrupted plaintext. We confirmed this behaviorally: a wrong `TWOFA_RESCUE_PASS` produces
   `secretstream: message authentication failed`, not garbage output.
6. The output is the plaintext: `otpauth://totp/...` lines, one per account, separated by
   newlines. Not JSON, despite the original ask; this is Ente's actual decrypted format.

Steps 4 and 5 are a **pure-Go reimplementation**, not a call into libsodium. We initially used
`github.com/jamesruan/sodium` (a cgo binding to the real libsodium C library) specifically
to avoid hand-rolling this construction, since it has subtle details that are easy to get
wrong against real backup data. That held true; we did get one detail wrong on the first
pure-Go pass (see Verification). But the cgo dependency meant the binary could only be
built on a machine with libsodium installed for the *target* platform, which broke
cross-compilation (e.g. `darwin/arm64` from an `amd64` host). We ported `pull()`,
`init_pull()`, and `rekey()` directly from libsodium's C source
(`crypto_secretstream/xchacha20poly1305/secretstream_xchacha20poly1305.c`) to
`golang.org/x/crypto/chacha20` (for `HChaCha20` and the raw keystream) and
`golang.org/x/crypto/poly1305` (the raw MAC primitive, not the higher-level
`chacha20poly1305` AEAD, since secretstream's per-chunk framing isn't a plain AEAD call).
The code lives in `crypt.go`.

## Verification

- Cross-checked our implementation line-for-line against Ente's own official
  `crypt.go`/`decrypt.go` reference tool: same libraries, same parameter handling, same
  Argon2id memory-unit conversion.
- Confirmed the export JSON the user actually had (`ente-auth-codes-2026-08-03.txt`) matches
  this exact schema (`version`, `kdfParams{memLimit,opsLimit,salt}`, `encryptedData`,
  `encryptionNonce`).
- When porting from cgo to pure Go, verified each layer independently rather than trusting
  a manual read of the C source:
  - Compared HChaCha20 subkey derivation and raw ChaCha20 keystream blocks byte-for-byte
    against a small cgo program calling libsodium's `crypto_core_hchacha20` and
    `crypto_stream_chacha20_ietf*` directly; these matched immediately.
  - The full `pull()` MAC check did *not* match on the first pass: a message encrypted by
    the real cgo library (via `github.com/jamesruan/sodium`) failed to authenticate under
    the new pure-Go code. Mirrored `pull()`'s exact MAC computation in a small C-via-cgo
    program to confirm the algorithm itself (as read from libsodium's source) was
    understood correctly (it was, and its output matched the real stored MAC), which
    narrowed the bug to the Go translation. That's how the ciphertext-padding quirk
    documented above was found: the Go code used the "correct" padding formula instead of
    libsodium's actual (intentionally quirky, self-admitted-as-wrong-in-source) one.
  - After the fix, round-tripped ciphertext generated by the real cgo library through the
    pure-Go decryptor across several message lengths (0, 16, 32, and 17 bytes, plus a
    ~1.4KB unaligned payload), all matched exactly, and confirmed a corrupted key still
    fails authentication cleanly rather than returning garbage.
- Successfully decrypted the user's real export with the pure-Go implementation and got
  correct, live TOTP codes.

## Where this lives in code

- `crypt.go`: `deriveArgonKey` (Argon2id) and the pure-Go secretstream decryption
  (`decryptChaCha20poly1305`, `secretStreamState`, and its `pull`/`rekey` methods).
- `main.go`: decrypts fully in memory and pipes straight into TOTP code generation; no
  plaintext ever touches disk unless `--decrypt -o <file>` is explicitly used.
