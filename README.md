# Ephemeral's Waybar Modules

Simple helper binary for my waybar configuration.

## Current supported modules

-   Lyrics: Shows the lyrics of currently playing song. (only for Spotify)
-   Pipewire: Simple volume controller for pipewire. (can be a shell script)
-   Weather: Shows weather :p

If you any cool idea of module, please create an issue I'll try my best to implement it.

## Install

Prerequisite: `go`

```sh
git clone https://github.com/Nadim147c/EWM
cd EWM

# To install of /usr/local/bin
make install

# To install in specific prefix (if prefix is /usr/ then install on /usr/bin/)
make install PREFIX=/usr/
```

## Usages

To find all the modules use `ewmod --help`. Here how to setup lyrics modules.

```
$ ewmod lyrics --init

Put the following object in your waybar config.

"custom/lyrics": {
    ... module example config
},
```

You can put this on your waybar `config.jsonc`

#### Logging

By default, EWM logs to `stderr`. You can export `EWM_LOG_FILE` to save log into a specific file.
