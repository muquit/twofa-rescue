# Encrypting files

This is a feature I added for myself, you may or may not need it. This can be
useful if do no use @ENTE_AUTH@ but you know your 2FA secrets and want to 
create the encrypted JSON file similar to the one exported by @ENTE_AUTH@ 
app. Please look at the FAQ.

Encrypt an input file, text or binary, into @ENTE_AUTH@'s JSON export
format, using the same Argon2id parameters @ENTE_AUTH@'s own app uses.
It works on any file, not just `otpauth://` lines. If the input is
`otpauth://` lines, the result is importable into the real @ENTE_AUTH@
app too, not just this tool.
```
twofa-rescue --encrypt -o encrypted.json plain.txt
```
**WARNING:** Anyone with the encrypted file and the password can decrypt
it. Treat `TWOFA_RESCUE_PASS` and the encrypted output with the same care
as the original secrets.
