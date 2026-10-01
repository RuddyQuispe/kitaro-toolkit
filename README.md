# kitaro-toolkit

"Kitaro Toolkit" is a desktop app for formatting text and converting it from one format to another.

## Live Development

Run `wails dev` in the project directory. This starts a Vite dev server with hot reload for the
frontend. A dev server also runs on http://localhost:34115 so you can call the Go methods from the
browser devtools.

## Building

Run `wails build` to build a redistributable production package.

On Linux distributions that only ship webkit2gtk-4.1 (e.g. Ubuntu 24.04+), add the build tag:

```sh
wails build -tags webkit2_41
```
