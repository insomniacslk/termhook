# termhook

`termhook` is a small library that attaches to a terminal, serial console or
other similar device, and lets you attach your own hook on terminal output.

## Example

See [cmds/termhook/main.go](cmds/termhook/main.go) for a simple terminal
emulator built with the termhook library.

Warning: at the moment there is no way to exit the terminal without killing the
running process, but will be implemented soon.
