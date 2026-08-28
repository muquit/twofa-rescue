# Testing

## Running tests

```
make test
```

This runs the full test suite. It takes about 10 seconds because one test uses
the real @ENTE_AUTH@ Argon2id parameters.

You can also run the tests directly:

```
go test ./...
```

`crypt_test.go` tests the pure @GO@ implementation of
`crypto_secretstream_xchacha20poly1305` against output created by the real
@LIBSODIUM@ C library.

It checks:

* Ciphertext created by the real @LIBSODIUM@ library can be decrypted.
* Encrypting the same test data produces the expected @LIBSODIUM@ output.
* Encryption and decryption work with different message sizes.
* A wrong password is rejected.
* Corrupted ciphertext is rejected.
* The complete `--encrypt` and `--decrypt` commands work with a file on disk.

The tests found a padding bug while the code was being converted from cgo to
pure @GO@. The bug only affected messages whose length was not a multiple of 16
bytes.

See [ente-auth-export-encryption-algorithm-claude.md](https://github.com/muquit/twofa-rescue/blob/main/docs/ente-auth-export-encryption-algorithm-claude.md)
for details.
