# HarnezPad interface

This Vite + React application is the complete HarnezPad interface hosted by the
Native SDK macOS shell. It uses the shadcn Base UI Luma preset `b2D0wqNxT`,
Geist typography, and the preset's Hugeicons library.

The production build is fully local and is packaged under
`HarnezPad.app/Contents/Resources/frontend/dist`. Setup and gateway guidance
live in Settings and the onboarding dialog; there is no embedded Help WebView.

```bash
pnpm install --frozen-lockfile
pnpm lint
pnpm test
pnpm build
```

Inspect the current preset and installed components with:

```bash
pnpm dlx shadcn@latest info --json
```

Use the shadcn CLI to add or inspect components; do not copy registry source by
hand.
