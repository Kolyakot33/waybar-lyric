# WayTune

A collection of custom waybar modules.

## Current supported modules

-   Lyrics: Shows the lyrics of currently playing song. (only for Spotify)
-   Pipewire: Simple volume controller for pipewire. (can be a shell script)
-   Weather: Shows weather :p

If you any cool idea of module, please create an issue I'll try my best to implement it.

## Install

Prerequisite: `go`

```sh
git clone https://github.com/Nadim147c/WayTune
cd WayTune

echo "Compiling..."
make
```

```sh
make install
```

```sh
# To install in specific prefix
make install PREFIX=/usr
```

## Usages

To find all the modules use `waytune --help`. Here how to setup lyrics modules.

```
$ waytune lyrics --init

Put the following object in your waybar config.

"custom/lyrics": {
    ... module example config
},
```

You can put this on your waybar `config.jsonc`

#### Logging

By default, WayTune logs to `stderr`. You can export `WAYTUNE_LOG_FILE` to save log into a specific file.
