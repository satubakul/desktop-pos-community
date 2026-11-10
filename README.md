# Kasir Kita

Kasir Kita is a local desktop POS application for managing multiple businesses in one installation.

## Current implementation

The first implementation slice includes:

- A Welcome screen and business onboarding form.
- A primary profile and optional secondary profiles for each business.
- Creation of the first outlet.
- Local SQLite persistence and workspace switching.
- A business home screen with a Cashier entry point.

Checkout, catalog management, and inventory screens are not implemented yet.

## Run locally

Requirements:

- Go 1.27 or later.
- A C compiler and the native desktop development libraries required by Fyne on the build OS.

Run the desktop app from the repository root:

```sh
go run ./cmd/kasir-kita
```

The SQLite database is created under the operating system's user config directory in `KasirKita/kasir-kita.sqlite`.

The Go module path is `github.com/satubakul/desktop-pos-community`. The source layout keeps application packages under `internal/` and does not use a top-level `pkg/` directory.
