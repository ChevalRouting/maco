# Installable web app

The web UI includes a web app manifest, standalone launch metadata, and the
existing maco icons. This first pass preserves the current UI and login behavior.
It is an online Home Screen web app, with no offline cache, service worker,
background command queue, or push notifications.

Apple Home Screen web apps do not require a service worker. See
[WebKit's explanation](https://webkit.org/blog/17333/webkit-features-in-safari-26-0/).
Offline handling and browser-specific install enhancements can be added later
as a separate scope.

## Installation

1. Build the embedded UI with `task build-ui` and run the maco server.
2. Open the server's HTTPS address in Safari and sign in.
3. On iPad, use Share, then Add to Home Screen. Enable Open as Web App if offered.
4. Launch maco from the new icon. Sign in again if the standalone session does
   not share the browser session.

Use an address reachable from the device. `localhost` on the iPad means the
iPad itself, not the Mac server. The server's default self-signed certificate
needs an appropriate trust setup, or replace it with a certificate trusted by
the device. This change does not configure remote access or certificate trust.

The existing JWT remains in sessionStorage and expires after 24 hours. This
change does not add persistent sign-in. Closing/relaunching the web app can
require login again. An available server and network connection are required.

## Validation scope

Desktop checks: production build, manifest and icon paths in the build output,
and the embedded server's static asset and SPA routing checks. No physical iPad
validation has been performed.

Deferred until an iPad is available: Home Screen installation and launch,
portrait/landscape and multitasking layouts, software/external keyboard input,
graphical and serial console touch behavior, and sleep/network reconnection.
These checks should inform a dedicated UI pass rather than be assumed complete.
