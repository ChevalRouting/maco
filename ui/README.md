# maco-ui

The React + Tailwind component library behind maco's web UI, styled after macOS
and iOS: system font and colors, translucent sidebar, inset grouped lists,
segmented controls, pop-up buttons, sheets and action sheets. Animations only
use transform and opacity.

The app in `../web` depends on it as `maco-ui` (`file:../ui`) and imports the
built `dist/`, so build the library first:

```bash
npm ci
npm run build        # -> dist/
npm run typecheck
npm run check:style
```

From the repository root, `task ui-build` runs `npm ci` and the build, and
`task web-build` and `task web-dev` run it for you.
