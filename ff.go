package evdev

import (
	"encoding/binary"
	"fmt"
	"time"
	"unsafe"
)

type EffectID int16

type EffectType uint16
const (
	RumbleEffect EffectType = FF_RUMBLE
)

type FFTriggerC struct {
	button uint16
	interval uint16
}

type FFReplayC struct {
	length uint16
	delay uint16
}

type FFEffectC struct {
	effectType EffectType
	id EffectID
	direction uint16
	trigger FFTriggerC
	replay FFReplayC
	// The union is 4-byte aligned due to the u32 in ff_periodic_effect, so
	// we need two bytes of padding here.
	// In general, the ioctl arguments are a total mess in terms of packing
	_pad [2]byte
}

type FFRumbleEffect struct {
	FFEffectC
	strongMagnitude uint16
	weakMagnitude uint16
}

func (dev *InputDevice) SupportsFFRumble() bool {
	ff, ok := dev.Capabilities[CapabilityType{ Type: EV_FF, Name: "EV_FF" } ]
	if !ok {
		return false
	}

	for _, ev := range ff {
		if ev.Code == FF_RUMBLE {
			return true
		}
	}

	return false
}

type FFEffect struct {
	dev *InputDevice
	id EffectID
}

func (dev *InputDevice) CreateFFRumbleEffect(strongMag, weakMag float32, duration time.Duration) (*FFEffect, error) {
	if strongMag > 1.0 || strongMag < 0.0 {
		return nil, fmt.Errorf("Strong magnitude out of range.")
	}

	if weakMag > 1.0 || weakMag < 0.0 {
		return nil, fmt.Errorf("Weak magnitude out of range.")
	}

	ms := uint16(duration / time.Millisecond)
	if  duration > 0 && ms == 0 {
		return nil, fmt.Errorf("Duration too short.")
	}
	if ms > 32767 {
		return nil, fmt.Errorf("Duration too long.")
	}

	effect := FFRumbleEffect{
		FFEffectC: FFEffectC{
			effectType: RumbleEffect,
			id: -1,
			replay: FFReplayC {
				length: ms,
				delay: 0,
			},
		},
		strongMagnitude: uint16(strongMag * 0xffff),
		weakMagnitude: uint16(weakMag * 0xffff),
	}

	errno := ioctl(dev.File.Fd(), uintptr(EVIOCSFF), unsafe.Pointer(&effect))
	if errno != 0 {
		return nil, error(errno)
	}

	return &FFEffect{ dev: dev, id : effect.id }, nil
}

func (eff *FFEffect) Delete() error {
	errno := ioctl(eff.dev.File.Fd(), uintptr(EVIOCRMFF), unsafe.Pointer(&eff.id))
	if errno != 0 {
		return error(errno)
	}
	return nil
}

func (eff *FFEffect) Play() error {
	ev := InputEvent {
		Type: EV_FF,
		Code: uint16(eff.id),
		Value: 1,
	}
	return binary.Write(eff.dev.File, binary.LittleEndian, ev)
}

func (eff *FFEffect) Stop() error {
	ev := InputEvent {
		Type: EV_FF,
		Code: uint16(eff.id),
		Value: 0,
	}
	return binary.Write(eff.dev.File, binary.LittleEndian, ev)
}
