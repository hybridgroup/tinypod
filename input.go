//go:build esp32c3 || rp2350

package main

import (
	"encoding/binary"
	"machine"
	"time"

	"tinygo.org/x/drivers/seesaw"
)

type event uint8

const (
	none event = iota
	up
	down
	left
	right
	selectItem
	back
	playPause
	volumeUp
	volumeDown
)

type input interface {
	poll() event
}

// Adafruit Mini I2C Gamepad QT pin numbers, from
// https://learn.adafruit.com/gamepad-qt/arduino
const (
	gamepadAddr  = 0x50
	gamepadX     = 6
	gamepadY     = 2
	gamepadA     = 5
	gamepadB     = 1
	gamepadSel   = 0
	gamepadStart = 16
	gamepadJoyX  = 14
	gamepadJoyY  = 15
	gamepadMask  = 1<<gamepadX | 1<<gamepadY | 1<<gamepadA | 1<<gamepadB | 1<<gamepadSel | 1<<gamepadStart
)

const (
	repeatDelay = 400 * time.Millisecond
	repeatRate  = 120 * time.Millisecond
)

type gamepad struct {
	ss      *seesaw.Device
	buttons uint32
	held    event
	next    time.Time
	menu    *button
}

// newGamepad returns nil if no gamepad answers on the bus.
func newGamepad(menu *button) *gamepad {
	ss := seesaw.New(i2c)
	ss.Address = gamepadAddr
	ss.ReadDelay = 2 * time.Millisecond
	if _, err := ss.ReadRegister(seesaw.ModuleStatusBase, seesaw.FunctionStatusHwId); err != nil {
		return nil
	}
	var mask [4]byte
	binary.BigEndian.PutUint32(mask[:], gamepadMask)
	ss.Write(seesaw.ModuleGpioBase, seesaw.FunctionGpioDirclrBulk, mask[:])
	ss.Write(seesaw.ModuleGpioBase, seesaw.FunctionGpioPullenset, mask[:])
	ss.Write(seesaw.ModuleGpioBase, seesaw.FunctionGpioBulkSet, mask[:])
	return &gamepad{ss: ss, menu: menu}
}

func (g *gamepad) poll() event {
	if g.menu.poll() != none {
		return back
	}
	var buf [4]byte
	if g.ss.Read(seesaw.ModuleGpioBase, seesaw.FunctionGpioBulk, buf[:]) != nil {
		return none
	}
	pressed := ^binary.BigEndian.Uint32(buf[:]) & gamepadMask
	edge := pressed &^ g.buttons
	g.buttons = pressed
	switch {
	case edge&(1<<gamepadA) != 0:
		return selectItem
	case edge&(1<<gamepadB) != 0, edge&(1<<gamepadSel) != 0:
		return back
	case edge&(1<<gamepadStart) != 0:
		return playPause
	case edge&(1<<gamepadX) != 0:
		return volumeUp
	case edge&(1<<gamepadY) != 0:
		return volumeDown
	}
	return g.stick()
}

// stick turns the joystick into up, down, left and right with auto repeat.
func (g *gamepad) stick() event {
	x, errx := g.analog(gamepadJoyX)
	y, erry := g.analog(gamepadJoyY)
	if errx != nil || erry != nil {
		return none
	}
	dir := none
	switch {
	case y < 256:
		dir = up
	case y > 768:
		dir = down
	case x < 256:
		dir = right
	case x > 768:
		dir = left
	}
	now := time.Now()
	switch {
	case dir == none:
	case dir != g.held:
		g.next = now.Add(repeatDelay)
	case now.After(g.next):
		g.next = now.Add(repeatRate)
	default:
		return none
	}
	g.held = dir
	return dir
}

func (g *gamepad) analog(pin byte) (int, error) {
	var buf [2]byte
	err := g.ss.Read(seesaw.ModuleAdcBase, seesaw.FunctionAdcChannelOffset+seesaw.FunctionAddress(pin), buf[:])
	return int(binary.BigEndian.Uint16(buf[:])), err
}

const (
	longPress     = 500 * time.Millisecond
	veryLongPress = 1500 * time.Millisecond
)

// button is the expansion board button on D1. Alone it does short press
// down, long press select and very long press back.
type button struct {
	pin   machine.Pin
	down  bool
	since time.Time
}

func newButton() *button {
	b := &button{pin: machine.D1}
	b.pin.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	return b
}

func (b *button) poll() event {
	pressed := !b.pin.Get()
	now := time.Now()
	switch {
	case pressed && !b.down:
		b.down, b.since = true, now
	case !pressed && b.down:
		b.down = false
		switch held := now.Sub(b.since); {
		case held >= veryLongPress:
			return back
		case held >= longPress:
			return selectItem
		case held > 20*time.Millisecond:
			return down
		}
	}
	return none
}
