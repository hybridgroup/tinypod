package tinypod

import (
	"encoding/binary"
	"time"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/seesaw"
)

// Event is a user action.
type Event uint8

const (
	None Event = iota
	Up
	Down
	Left
	Right
	Select
	Back
	PlayPause
	VolumeUp
	VolumeDown
	Next
	Previous
)

// Input reports at most one event each time tinypod polls it, about every
// 30 ms.
type Input interface {
	Poll() Event
}

var now = time.Now

const (
	holdTime      = 500 * time.Millisecond
	longHoldTime  = 1500 * time.Millisecond
	repeatDelay   = 400 * time.Millisecond
	repeatRate    = 120 * time.Millisecond
	debounceDelay = 20 * time.Millisecond
)

// Button is one button. Pressed reports if it is down, for example
// func() bool { return !pin.Get() } for a button to GND with a pull up.
//
// Without Hold, Press fires when the button goes down and, with Repeat, again
// while it stays down. With Hold, Press fires on a short press when the button
// comes up and Hold fires once the button is down for 500 ms.
type Button struct {
	Pressed func() bool
	Press   Event
	Hold    Event
	Repeat  bool
}

type buttonState struct {
	down  bool
	held  bool
	since time.Time
	next  time.Time
}

type buttons struct {
	bs []Button
	st []buttonState
}

// Buttons returns an Input for a set of buttons.
func Buttons(bs ...Button) Input {
	return &buttons{bs: bs, st: make([]buttonState, len(bs))}
}

func (b *buttons) Poll() Event {
	t := now()
	ev := None
	for i, btn := range b.bs {
		if e := b.st[i].update(btn, btn.Pressed(), t); e != None && ev == None {
			ev = e
		}
	}
	return ev
}

func (s *buttonState) update(b Button, pressed bool, t time.Time) Event {
	switch {
	case pressed && !s.down:
		s.down, s.held, s.since = true, false, t
		s.next = t.Add(repeatDelay)
		if b.Hold == None {
			return b.Press
		}
	case pressed && s.down:
		if b.Hold != None && !s.held && t.Sub(s.since) >= holdTime {
			s.held = true
			return b.Hold
		}
		if b.Hold == None && b.Repeat && !t.Before(s.next) {
			s.next = t.Add(repeatRate)
			return b.Press
		}
	case !pressed && s.down:
		s.down = false
		if b.Hold != None && !s.held && t.Sub(s.since) >= debounceDelay {
			return b.Press
		}
	}
	return None
}

type holdButton struct {
	pressed func() bool
	down    bool
	since   time.Time
}

// HoldButton lets one button do everything. A tap moves down, a 0.5 s hold
// selects and a 1.5 s hold goes back. It acts when the button comes up.
func HoldButton(pressed func() bool) Input {
	return &holdButton{pressed: pressed}
}

func (b *holdButton) Poll() Event {
	pressed := b.pressed()
	t := now()
	switch {
	case pressed && !b.down:
		b.down, b.since = true, t
	case !pressed && b.down:
		b.down = false
		switch held := t.Sub(b.since); {
		case held >= longHoldTime:
			return Back
		case held >= holdTime:
			return Select
		case held >= debounceDelay:
			return Down
		}
	}
	return None
}

type inputs []Input

// Inputs combines several inputs into one.
func Inputs(in ...Input) Input {
	return inputs(in)
}

func (in inputs) Poll() Event {
	ev := None
	for _, i := range in {
		if e := i.Poll(); e != None && ev == None {
			ev = e
		}
	}
	return ev
}

// GamepadMap gives the event for each control of the gamepad.
type GamepadMap struct {
	A, B, X, Y, Select, Start Event
	Up, Down, Left, Right     Event
}

// DefaultGamepadMap moves with the joystick, opens with A and goes back with B.
var DefaultGamepadMap = GamepadMap{
	A: Select, B: Back, X: VolumeUp, Y: VolumeDown, Select: Back, Start: PlayPause,
	Up: Up, Down: Down, Left: Left, Right: Right,
}

// Adafruit Mini I2C Gamepad QT pin numbers, from
// https://learn.adafruit.com/gamepad-qt/arduino
const (
	GamepadAddr  = 0x50
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

type gamepad struct {
	ss      *seesaw.Device
	m       GamepadMap
	buttons uint32
	held    Event
	next    time.Time
}

// Gamepad returns an Adafruit Mini I2C Gamepad QT on bus, or nil if no
// gamepad answers.
func Gamepad(bus drivers.I2C, m GamepadMap) Input {
	ss := seesaw.New(bus)
	ss.Address = GamepadAddr
	ss.ReadDelay = 2 * time.Millisecond
	if _, err := ss.ReadRegister(seesaw.ModuleStatusBase, seesaw.FunctionStatusHwId); err != nil {
		return nil
	}
	var mask [4]byte
	binary.BigEndian.PutUint32(mask[:], gamepadMask)
	ss.Write(seesaw.ModuleGpioBase, seesaw.FunctionGpioDirclrBulk, mask[:])
	ss.Write(seesaw.ModuleGpioBase, seesaw.FunctionGpioPullenset, mask[:])
	ss.Write(seesaw.ModuleGpioBase, seesaw.FunctionGpioBulkSet, mask[:])
	return &gamepad{ss: ss, m: m}
}

func (g *gamepad) Poll() Event {
	var buf [4]byte
	if g.ss.Read(seesaw.ModuleGpioBase, seesaw.FunctionGpioBulk, buf[:]) != nil {
		return None
	}
	pressed := ^binary.BigEndian.Uint32(buf[:]) & gamepadMask
	edge := pressed &^ g.buttons
	g.buttons = pressed
	for _, b := range [...]struct {
		pin uint
		ev  Event
	}{
		{gamepadA, g.m.A}, {gamepadB, g.m.B}, {gamepadSel, g.m.Select},
		{gamepadStart, g.m.Start}, {gamepadX, g.m.X}, {gamepadY, g.m.Y},
	} {
		if edge&(1<<b.pin) != 0 && b.ev != None {
			return b.ev
		}
	}
	return g.stick()
}

// stick turns the joystick into direction events with auto repeat.
func (g *gamepad) stick() Event {
	x, errx := g.analog(gamepadJoyX)
	y, erry := g.analog(gamepadJoyY)
	if errx != nil || erry != nil {
		return None
	}
	dir := None
	switch {
	case y < 256:
		dir = g.m.Up
	case y > 768:
		dir = g.m.Down
	case x < 256:
		dir = g.m.Right
	case x > 768:
		dir = g.m.Left
	}
	t := now()
	switch {
	case dir == None:
	case dir != g.held:
		g.next = t.Add(repeatDelay)
	case !t.Before(g.next):
		g.next = t.Add(repeatRate)
	default:
		return None
	}
	g.held = dir
	return dir
}

func (g *gamepad) analog(pin byte) (int, error) {
	var buf [2]byte
	err := g.ss.Read(seesaw.ModuleAdcBase, seesaw.FunctionAdcChannelOffset+seesaw.FunctionAddress(pin), buf[:])
	return int(binary.BigEndian.Uint16(buf[:])), err
}
