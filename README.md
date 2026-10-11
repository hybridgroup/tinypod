# tinypod

<img src="images/logo.svg" alt="tinypod logo" width="160">

tinypod is an iPod classic style music player that runs on real hardware using [TinyGo](https://tinygo.org). 

tinypod uses [tinyplayer](https://github.com/hybridgroup/tinyplayer) to play audio using WAV files. It works with several different displays, can play songs on an SD card or in flash, and can use a configurable set of buttons based on your specific needs.

Here are a few built-in configurations:

| Device | Folder | Display | Songs | Input | Audio |
|---|---|---|---|---|---|
| xiao-esp32c3 or xiao-rp2350 on the Seeed XIAO expansion board | `devices/xiao-expansion` | SSD1306 128x64 | microSD card | Gamepad QT, or the board button | I2S DAC |
| Pico 2 | `devices/pico2-ssd1306` | SSD1306 128x64 | flash, USB drive | 4 buttons | I2S DAC |
| Gopher badge | `devices/gopher-badge` | ST7789 320x240 color | flash, USB drive | 6 badge buttons | speaker |

## Build and flash

```
tinygo flash -tags fat_noexfat -target=xiao-rp2350 ./devices/xiao-expansion
tinygo flash -tags fat_noexfat -target=xiao-esp32c3 ./devices/xiao-expansion
tinygo flash -tags fat_noexfat -target=pico2 ./devices/pico2-ssd1306
tinygo flash -tags fat_noexfat -target=gopher-badge ./devices/gopher-badge
```

The serial output shows the number of songs and albums discovered by tinypod.

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

The menus and Now Playing use these controls. Each device maps its buttons to them.

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

## xiao-expansion

Folder `devices/xiao-expansion`.

A xiao-esp32c3 or xiao-rp2350 plugged into the [Seeed XIAO expansion board](https://wiki.seeedstudio.com/Seeeduino-XIAO-Expansion-Board/). The expansion board has the microSD slot, the OLED, a button and a buzzer. You connect the DAC and the gamepad.

Parts:

- xiao-esp32c3 or xiao-rp2350
- Seeed XIAO expansion board
- An I2S DAC such as the Adafruit PCM5102, with headphones or a powered speaker
- Optional: an [Adafruit Mini I2C Gamepad QT](https://www.adafruit.com/product/5743) and a Grove to STEMMA QT cable
- A microSD card, see [SD card](#sd-card)

The expansion board uses D1 for the button, D2 and D8 to D10 for the SD card, D3 for the buzzer, and D4/D5 for I2C. The DAC uses the free pins D0, D6 and D7. Use jumper wires from the expansion board pin headers.

| DAC pin | XIAO pin | xiao-esp32c3 | xiao-rp2350 |
|---|---|---|---|
| VIN | 3V3 | | |
| GND | GND | | |
| BCK | D6 | GPIO21 | GPIO0 |
| WSEL (LCK) | D7 | GPIO20 | GPIO1 |
| DIN | D0 | GPIO2 | GPIO26 |

On the RP2350 the WSEL pin must be the pin after BCK, because of how the PIO I2S program works. See [AUDIO.md in tinyplayer](https://github.com/hybridgroup/tinyplayer/blob/main/AUDIO.md) for the extra pins that some PCM5102 boards have.

Plug the gamepad into the Grove I2C port. It shares the bus with the OLED. The gamepad uses address 0x50 and the OLED uses 0x3C.

With the gamepad:

| Control | Event |
|---|---|
| Joystick | Up, Down, Left, Right |
| A | Select |
| B, Select | Back |
| Start | PlayPause |
| X, Y | VolumeUp, VolumeDown |
| Expansion board button | Back |

With no gamepad, the expansion board button does everything. It acts when you release it.

| Press | Event |
|---|---|
| Tap | Down |
| Hold 0.5 seconds | Select |
| Hold 1.5 seconds | Back |

tinypod looks for the gamepad only at start. To change from one input to the other, reset the board.

## pico2-ssd1306

Folder `devices/pico2-ssd1306`.

A Pico 2 with an SSD1306 OLED, four buttons and an I2S DAC. Songs are in flash, which shows up as a USB drive. Power and flashing use the Pico 2 USB port.

Parts:

- Raspberry Pi Pico 2
- SSD1306 128x64 I2C OLED at address 0x3C
- 4 push buttons
- An I2S DAC such as the Adafruit PCM5102, with headphones or a powered speaker

The physical pin is the Pico 2 header pin. Pin 1 is next to the USB port on the left side.

OLED on I2C1:

| OLED pin | Pico 2 | Physical pin |
|---|---|---|
| VCC | 3V3(OUT) | 36 |
| GND | GND | 8 |
| SDA | GPIO6 | 9 |
| SCL | GPIO7 | 10 |

Most OLED modules have pull up resistors for I2C. If yours uses address 0x3D, change the address in `NewSSD1306` in `main.go`.

DAC:

| DAC pin | Pico 2 | Physical pin |
|---|---|---|
| VIN | 3V3(OUT) | 36 |
| GND | GND | 3 |
| DIN | GPIO2 | 4 |
| BCK | GPIO3 | 5 |
| WSEL (LCK) | GPIO4 | 6 |

WSEL must be the pin after BCK, because of how the PIO I2S program works. If your DAC board has more pins than the Adafruit one, connect SCK to GND, XSMT to 3V3, and FMT, FLT and DEMP to GND. See [AUDIO.md in tinyplayer](https://github.com/hybridgroup/tinyplayer/blob/main/AUDIO.md).

Buttons. Connect one leg of each button to its pin and the other leg to GND. The pins use the internal pull up, so no resistors are necessary.

| Button | Pico 2 | Physical pin | GND near it |
|---|---|---|---|
| Up | GPIO10 | 14 | 13 |
| Down | GPIO11 | 15 | 13 or 18 |
| Select | GPIO12 | 16 | 18 |
| Back | GPIO13 | 17 | 18 |

| Button | Press | Hold 0.5 seconds |
|---|---|---|
| Up | Up, repeats while held | |
| Down | Down, repeats while held | |
| Select | Select | PlayPause |
| Back | Back | Next |

The I2C pins are not the default GPIO4 and GPIO5, because GPIO4 is the I2S WSEL pin.

If the computer does not see the Pico 2 at all, hold BOOTSEL while you plug in the USB cable. A drive named RP2350 shows up and `tinygo flash` can write to it.

## gopher-badge

Folder `devices/gopher-badge`.

The gopher badge needs no wiring. tinypod uses the 320x240 color screen, the six buttons and the speaker.

| Button | Press | Hold 0.5 seconds |
|---|---|---|
| Up | Up, repeats while held | |
| Down | Down, repeats while held | |
| Left | Left | |
| Right | Right | |
| A | Select | PlayPause |
| B | Back | |

The speaker plays mono audio through the `pwm` package.

## Add a device

Write a main that sets up the hardware and fills a `tinypod.Config`:

```go
err := tinypod.Run(tinypod.Config{
	Display: disp,                // any driver with Size, SetPixel and Display
	Theme:   tinypod.MonoTheme(), // or ColorTheme, or your own font and colors
	Output:  out,                 // I2S or pwm.PWM
	Volume:  vol,                 // from MountSD or MountFlash
	Input: tinypod.Buttons(
		tinypod.Button{Pressed: up, Press: tinypod.Up, Repeat: true},
		tinypod.Button{Pressed: sel, Press: tinypod.Select, Hold: tinypod.PlayPause},
	),
})
```

- Display: tinypod uses `FillRectangle` too if the driver has it. For an SSD1306 on I2C use `tinypod.NewSSD1306`, see [Screen](#screen).
- Theme: the layout comes from the display size and the font. `MonoTheme` uses an 8 point font for small screens. `ColorTheme` uses Noto Sans 12 point.
- Volume: `MountSD` needs the SPI bus at 400 kHz and a function that raises the clock after setup. `MountFlash` takes a tinyplayer `flashdisk.Disk`. Call `msc.Port(disk)` before `machine.USBDev.Configure` so it shows up as a USB drive.
- Input: `Buttons` for buttons, `HoldButton` for one button, `Gamepad` for the Gamepad QT with a `GamepadMap`, and `Inputs` to use several at once. `Pressed` is a function, so a button can come from a pin, an I/O expander or a touch pad.
- ReadAhead: set it if the display or the storage holds the CPU for long, as on the ESP32-C3. See [Read ahead](#read-ahead).

## How it works

tinypod consists of four main subsystems:

- Display
Display shows what is going on to the user of the tinypod.

- Input
Input reads whatever buttons are connected so that the user can operate the tinypod.

- Library
Library provides access to the actual music on the tinypod. It uses the Storage interface to be able to find albums anbd songs on SD card or flash memory.

- Deck
Deck is what actually plays back the music so the user can hear it.

## Known issues

- tinypod on an SD card needs the `sd` fix in [tinygo-org/drivers#905](https://github.com/tinygo-org/drivers/pull/905). Without it, the SD card does not mount.
- On the RP2350 with piolib I2S and SPI it needs [tinygo-org/pio#65](https://github.com/tinygo-org/pio/pull/65). Without it, piolib I2S and SPI0 use the same DMA channel, and the board stops when a song starts.
- `go.mod` uses the branches of these two PRs until they are merged.
- The `sd` package reads several blocks at a time correctly only on SDHC cards. Do not use cards of 2 GB or less.
- Cards of 64 GB or more usually come formatted as exFAT. Format them as FAT32.
- The ColorTheme font is Noto Sans from tinyfont, under the SIL Open Font License.

## License

Apache 2.0, see [LICENSE](LICENSE).
