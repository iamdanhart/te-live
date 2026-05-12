# QR Code Generation

Requires `APP_PUBLIC_URL` or `APP_URL` set in `.env`. Uses `APP_PUBLIC_URL` if set, falls back to `APP_URL`.

Generate an SVG (scales to any print size):

```bash
source .env && qrencode -t SVG -o docs/qr.svg "$APP_URL"
```

Or display in terminal:

```bash
source .env && qrencode -t ANSIUTF8 "$APP_URL"
```

Install if needed:

```bash
brew install qrencode
```