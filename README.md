# Ephemeral's Waybar Modules

Simple helper binary for my waybar configuration.

#### Current supported modules

-   Lyrics: Shows the lyrics of currently playing song. (only for Spotify)
-   Pipewire: Simple volume controller for pipewire. (can be a shell script)

#### Install

```sh
git clone https://github.com/Nadim147c/EWM
cd EWM

# To install of /usr/local/bin
make install

# To install in go path
go install

# To install in specific prefix (if prefix is /usr/ then install on /usr/bin/)
make install PREFIX=/usr/
```
