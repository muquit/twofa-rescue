## Installing using Homebrew on Mac/Linux

You will need to install [Homebrew](https://brew.sh/) first.

### Install

First install the custom tap, then trust it. Homebrew 6.0+ refuses to load
formulae from third-party taps until they are explicitly trusted.

```
brew tap muquit/ente-totp-cli https://github.com/muquit/ente-totp-cli.git
brew trust muquit/ente-totp-cli
brew install twofa-rescue
```

Or tap, trust and install in one go:
```
brew tap muquit/ente-totp-cli https://github.com/muquit/ente-totp-cli.git
brew trust muquit/ente-totp-cli
brew install muquit/ente-totp-cli/twofa-rescue
```

### Upgrade
```
brew upgrade twofa-rescue
```

### Uninstall
```
brew uninstall twofa-rescue
```

### Remove the tap
```
brew untap muquit/ente-totp-cli
```
