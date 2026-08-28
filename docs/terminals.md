# Tested Terminals

The following terminals displayed scannable QR codes in my tests. Other
terminals may work as well. If you test another terminal successfully, create
an issue and I will add it to the list.


|Terminal |OS |Result |Rendering|
|---|---|---|---|
| @ITERM2@ | macOS | ✅ Works | @ITERM2_IMAGE@ |
| @KITTY@ | macOS, Ubuntu 24.04 | ✅ Works | Unicode block fallback |
| @GHOSTTY@ | macOS | ✅ Works | Unicode block fallback |
| @WEZTERM@ | macOS, Ubuntu, Windows | ✅ Works | Unicode block fallback |
| Apple Terminal | macOS | ✅ Works (auto-resizes the window if needed, see [FAQ](#faq)) | Solid ANSI block fill (custom) |
| Windows Terminal | Windows 11 Pro | ✅ Works | Unicode block fallback |
| GNOME Terminal | Ubuntu 24.04 | ✅ Works | Unicode block fallback |
| xterm | Ubuntu 24.04 | ✅ Works | Unicode block fallback |
| mlterm | Ubuntu 24.04 | ✅ Works | Unicode block fallback |
| foot | Ubuntu 24.04 | ✅ Works | Unicode block fallback |
| konsole | Ubuntu 24.04 (GNOME) | ✅ Works | Unicode block fallback |


Apple Terminal gets its own dedicated renderer (solid ANSI background-color
blocks, no font glyph involved), since testing showed the usual Unicode
block fallback comes out distorted no matter which font is
selected.

Every other terminal above, including ones that natively
support @KITTY_GRAPHICS@ or @SIXEL@, gets that same Unicode
block-character fallback. Where it says "✅ Works" for those, that means
the fallback happened to render a scannable QR code on that terminal's
default font, not that the tool used that terminal's own graphics
protocol.

Run with `--debug` to see which path was actually taken.


The testing was done as follows:

* Display demo QR Code

```
twofa-rescue --demo-qr
```

* Import the secret from the QR code using an app such as @GOOGLE_AUTHENTICATOR@.
  After each successful import, delete the entry before testing it again in
  another terminal. I also tested other authenticator apps; see
  [Demo QR code](#demo-qr-code) for details.
