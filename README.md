# tinypod

<img src="images/logo.svg" alt="tinypod logo" width="160">

tinypod is an iPod classic style music player that runs on real hardware using [TinyGo](https://tinygo.org). 

tinypod uses [tinyplayer](https://github.com/hybridgroup/tinyplayer) to play audio using WAV files. It works with several different displays, can play songs on an SD card or in flash, and can use a configurable set of buttons based on your specific needs.

## Supported Devices

Here are a few built-in supported device configurations:

| Device | Folder | Display | Songs | Input | Audio |
|---|---|---|---|---|---|
| xiao-esp32c3 or xiao-rp2350 on the Seeed XIAO expansion board | `devices/xiao-expansion` | SSD1306 128x64 | microSD card | Gamepad QT, or the board button | I2S DAC |
| Pico 2 | `devices/pico2-ssd1306` | SSD1306 128x64 | flash, USB drive | 4 buttons | I2S DAC |
| Gopher badge | `devices/gopher-badge` | ST7789 320x240 color | flash, USB drive | 6 badge buttons | speaker |

See [DEVICES](./DEVICES.md) for more information and how to add your own.

## Build and flash

```
tinygo flash -tags fat_noexfat -target=xiao-rp2350 ./devices/xiao-expansion
tinygo flash -tags fat_noexfat -target=xiao-esp32c3 ./devices/xiao-expansion
tinygo flash -tags fat_noexfat -target=pico2 ./devices/pico2-ssd1306
tinygo flash -tags fat_noexfat -target=gopher-badge ./devices/gopher-badge
```

The serial output will show the number of songs and albums discovered by tinypod when it first starts up.

## Songs

Put WAV files in folders under `/Music`. Each folder is an album. Files in one level of subfolders are added to the album too. If there is no `/Music` folder, tinypod uses the root of the volume.

```
/Music/Album One/01 Song.wav
/Music/Album One/02 Song.wav
/Music/Album Two/Disc 1/01 Song.wav
/Music/Single.wav
```

The song name on the screen is the file name without `.wav`. The album name is the folder name.

Use ffmpeg to make the files. 16 bit PCM sounds best. IMA ADPCM files are 4 times smaller, which helps when songs are in flash.

```
ffmpeg -i in.mp3 -ac 2 -ar 44100 -c:a pcm_s16le out.wav
ffmpeg -i in.mp3 -ac 2 -ar 22050 -c:a adpcm_ima_wav out.wav
```

### SD card

Use a FAT32 microSD card of 4 GB to 32 GB (SDHC). tinypod mounts the card read only and never writes to it.

### Flash and USB drive

On boards with songs in flash, the board shows up as a USB drive named tinypod. The first time, tinypod formats the flash. Copy WAV files to the drive. One second after the copy ends, tinypod stops playback, reads the songs again and goes back to the main menu.

| Device | Flash | Space for songs | 22.05 kHz stereo ADPCM |
|---|---|---|---|
| Pico 2 | 4 MB | 3 MB | about 2.4 minutes |
| Gopher badge | 8 MB | 6 MB | about 4.8 minutes |

22.05 kHz stereo ADPCM uses about 22 KB per second. Mono ADPCM at 16 kHz uses 8 KB per second and gives about 6.5 and 13 minutes.

## Controls

tinypod uses these controls. Each device maps its buttons to them.

| Event | Menus | Now Playing |
|---|---|---|
| Up, Down | Move | Volume |
| Left | Back | Previous song, or start of song after 3 seconds |
| Right | Open | Next song |
| Select | Open | Play/pause |
| Back | Back | Back to menus |
| PlayPause | Play/pause | Play/pause |
| VolumeUp, VolumeDown | Volume | Volume |
| Next, Previous | Next or previous song | Next or previous song |

If nothing is playing, PlayPause plays all songs from the start.

## How it works

tinypod consists of four main subsystems:

- Display
Display shows what is going on to the user of the tinypod. See `display.go` and `splash.go`.

- Input
Input reads whatever buttons are connected so that the user can operate the tinypod. See `input.go`.

- Library
Library provides access to the actual music on the tinypod. It uses the Storage interface to be able to find albums and songs on SD card or flash memory. See `library.go` and `storage.go`.

- Deck
Deck is what actually plays back the music so the user can hear it. See `deck.go`.

`app.go` connects them. It reads events from Input, runs the menus, tells Deck what to play from Library, and draws on Display.

## Known issues

- tinypod on an SD card needs the `sd` fix in [tinygo-org/drivers#905](https://github.com/tinygo-org/drivers/pull/905). Without it, the SD card does not mount.
- `go.mod` uses the branch of this PR until it is merged.
- The `sd` package reads several blocks at a time correctly only on SDHC cards. Do not use cards of 2 GB or less.
- Cards of 64 GB or more usually come formatted as exFAT. Format them as FAT32.
- The ColorTheme font is Noto Sans from tinyfont, under the SIL Open Font License.

## License

Apache 2.0, see [LICENSE](LICENSE).
