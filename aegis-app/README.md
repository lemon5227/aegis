# README

## About

This is the official Wails React-TS template.

You can configure the project by editing `wails.json`. More information about the project settings can be found
here: https://wails.io/docs/reference/project-config

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.

## WSL2 One-Click Setup

From repository root, run:

```bash
./scripts/setup_wsl2_env.sh
```

If you also want it to auto-start the app after setup:

```bash
./scripts/setup_wsl2_env.sh --start
```

The script installs Linux dependencies, Go, Node.js (via nvm), Wails CLI, frontend npm packages, and prints the default `wails dev` command with P2P seed relay envs.
