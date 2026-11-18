# Home Screen web app

You can add the maco web UI to the Home Screen of an iPhone or iPad (or install
it from other browsers). It is an online web app: no offline mode, service
worker or push notifications, so it needs the server to be reachable.

## Install

1. Build with the embedded UI (`task build-ui`) and run the server.
2. On the device, open the server's HTTPS address in Safari and sign in. Use an
   address the device can reach: `localhost` on an iPad is the iPad.
3. Tap Share, then Add to Home Screen. Turn on Open as Web App if offered.
4. Open maco from the new icon. Sign in again if the app does not share the
   browser's session.

The device must trust the server's certificate. Either trust maco's
self-signed certificate on the device or serve a trusted one (see
[configuration.md](configuration.md#tls-certificates)).

The session token is stored in `localStorage` and lasts 24 hours, so closing
the app or restarting the server does not sign you out.

## Icons

`apple-touch-icon.png` (180 px) for iOS and maskable `icon-192.png` and
`icon-512.png` for other browsers are rendered from `icon.svg`. The touch icon
is inlined into `index.html` as a data URI, because iOS fetches Home Screen
icons outside the page and rejects a self-signed certificate.

## Touch behavior

- Pages do not zoom when you focus a field (`maximum-scale=1`, 16 px inputs on
  phones). The installed app also blocks pinch and double-tap zoom; in the
  browser, pinch zoom still works.
- Opening a sheet or dialog does not focus a field, so the keyboard stays
  closed until you tap one.
- The layout respects the safe areas (notch, home indicator, landscape edges).
- The graphical console switches between Fit and 100%; at 100% a drag pans the
  view. On touch screens a Keyboard button opens the system keyboard and sends
  what you type to the guest, with extra keys for Esc, Tab, the arrows and
  Ctrl+Alt+Del.
- The theme follows the system unless you pick one in Preferences.

## Not yet tested

Nothing has been checked on a physical iPad yet: Home Screen install and
launch, portrait and landscape and multitasking layouts, software and external
keyboards, console touch input, and reconnecting after sleep or a network
change.
