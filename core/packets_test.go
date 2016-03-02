// Copyright © 2016 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package core

import (
	"math/rand"
	"testing"

	"github.com/TheThingsNetwork/ttn/utils/pointer"
	"github.com/brocaar/lorawan"
	. "github.com/smartystreets/assertions"
)

func randBytes(n int) []byte {
	bytes := make([]byte, n)
	for i := range bytes {
		bytes[i] = byte(rand.Intn(255))
	}
	return bytes
}

func newEUI() lorawan.EUI64 {
	devEUI := [8]byte{}
	copy(devEUI[:], randBytes(8))
	return devEUI
}

func simplePayload() (payload lorawan.PHYPayload, devAddr lorawan.DevAddr, key lorawan.AES128Key) {
	copy(devAddr[:], randBytes(4))
	copy(key[:], randBytes(16))

	payload = newPayload(devAddr, []byte("PLD123"), key, key)
	return
}

func newPayload(devAddr lorawan.DevAddr, data []byte, appSKey lorawan.AES128Key, nwkSKey lorawan.AES128Key) lorawan.PHYPayload {
	uplink := true

	macPayload := lorawan.NewMACPayload(uplink)
	macPayload.FHDR = lorawan.FHDR{
		DevAddr: devAddr,
		FCtrl: lorawan.FCtrl{
			ADR:       false,
			ADRACKReq: false,
			ACK:       false,
		},
		FCnt: 1,
	}
	macPayload.FPort = 10
	macPayload.FRMPayload = []lorawan.Payload{&lorawan.DataPayload{Bytes: data}}

	if err := macPayload.EncryptFRMPayload(appSKey); err != nil {
		panic(err)
	}

	payload := lorawan.NewPHYPayload(uplink)
	payload.MHDR = lorawan.MHDR{
		MType: lorawan.ConfirmedDataUp,
		Major: lorawan.LoRaWANR1,
	}
	payload.MACPayload = macPayload

	if err := payload.SetMIC(nwkSKey); err != nil {
		panic(err)
	}

	return payload
}

func marshalUnmarshal(t *testing.T, input Packet) interface{} {
	a := New(t)

	binary, err := input.MarshalBinary()
	a.So(err, ShouldBeNil)

	gOutput, err := UnmarshalPacket(binary)
	a.So(err, ShouldBeNil)

	a.So(gOutput, ShouldHaveSameTypeAs, input)

	return gOutput
}

func TestInvalidMarshalBases(t *testing.T) {
	// Only err when Metadata Marshal returns err
}

func TestBaseMarshalUnmarshal(t *testing.T) {
	a := New(t)

	s := uint(123)
	mpkt := basempacket{metadata: Metadata{Size: &s}}

	payload, _, _ := simplePayload()
	rpkt := baserpacket{payload: payload}
	hpkt := basehpacket{
		appEUI: newEUI(),
		devEUI: newEUI(),
	}
	apkt := baseapacket{
		payload: []byte{0x01, 0x02, 0x03},
	}
	gpkt := basegpacket{metadata: []Metadata{
		Metadata{Size: &s},
	}}

	binmpkt, err1 := mpkt.Marshal()
	a.So(err1, ShouldBeNil)
	binrpkt, err2 := rpkt.Marshal()
	a.So(err2, ShouldBeNil)
	binhpkt, err3 := hpkt.Marshal()
	a.So(err3, ShouldBeNil)
	binapkt, err4 := apkt.Marshal()
	a.So(err4, ShouldBeNil)
	bingpkt, err5 := gpkt.Marshal()
	a.So(err5, ShouldBeNil)

	newmpkt := basempacket{}
	newrpkt := baserpacket{}
	newhpkt := basehpacket{}
	newapkt := baseapacket{}
	newgpkt := basegpacket{}

	_, err6 := newmpkt.Unmarshal(binmpkt)
	a.So(err6, ShouldBeNil)
	a.So(*newmpkt.Metadata().Size, ShouldEqual, s)

	_, err7 := newrpkt.Unmarshal(binrpkt)
	a.So(err7, ShouldBeNil)
	// a.So()

	_, err8 := newhpkt.Unmarshal(binhpkt)
	a.So(err8, ShouldBeNil)

	_, err9 := newapkt.Unmarshal(binapkt)
	a.So(err9, ShouldBeNil)

	_, erra := newgpkt.Unmarshal(bingpkt)
	a.So(erra, ShouldBeNil)
	a.So(*newgpkt.Metadata()[0].Size, ShouldEqual, s)

}

func TestInvalidUnmarshalBases(t *testing.T) {
	a := New(t)

	p := basempacket{}

	err1 := unmarshalBases(0x01, []byte{}, &p)
	a.So(err1, ShouldNotBeNil)

	err2 := unmarshalBases(0x01, []byte{0x02}, &p)
	a.So(err2, ShouldNotBeNil)

	err3 := unmarshalBases(0x01, []byte{0x01}, &p)
	a.So(err3, ShouldNotBeNil)
}

func TestInvalidUnmarshalPacket(t *testing.T) {
	a := New(t)
	_, err := UnmarshalPacket([]byte{})
	a.So(err, ShouldNotBeNil)
}

func TestPacket(t *testing.T) {
	a := New(t)
	input := basempacket{
		metadata: Metadata{
			Codr: pointer.String("4/6"),
		},
	}

	binary, _ := input.Marshal()

	output := basempacket{
		metadata: Metadata{},
	}

	_, err := output.Unmarshal(binary)
	a.So(err, ShouldBeNil)
}

func TestInvalidRPacket(t *testing.T) {
	a := New(t)

	// No MACPayload
	_, err1 := NewRPacket(lorawan.PHYPayload{}, []byte{}, Metadata{})
	a.So(err1, ShouldNotBeNil)

	// Not a MACPayload
	_, err2 := NewRPacket(lorawan.PHYPayload{
		MACPayload: &lorawan.JoinRequestPayload{},
	}, []byte{}, Metadata{})
	a.So(err2, ShouldNotBeNil)
}

func TestRPacket(t *testing.T) {
	a := New(t)

	payload, devAddr, _ := simplePayload()
	gwEUI := []byte{}
	copy(gwEUI[:], randBytes(8))

	input, _ := NewRPacket(payload, gwEUI, Metadata{})

	gOutput := marshalUnmarshal(t, input)

	output := gOutput.(RPacket)

	a.So(output.Payload(), ShouldResemble, payload)
	a.So(output.GatewayId(), ShouldResemble, gwEUI)
	a.So(output.Metadata(), ShouldResemble, Metadata{})
	outputDevEUI := output.DevEUI()
	a.So(outputDevEUI[4:], ShouldResemble, devAddr[:])

	// TODO: Different MTypes
}

func TestSPacket(t *testing.T) {
	// Nope
}

func TestInvalidBPacket(t *testing.T) {
	a := New(t)

	// No MACPayload
	_, err1 := NewBPacket(lorawan.PHYPayload{}, Metadata{})
	a.So(err1, ShouldNotBeNil)

	// Not a MACPayload
	_, err2 := NewBPacket(lorawan.PHYPayload{
		MACPayload: &lorawan.JoinRequestPayload{},
	}, Metadata{})
	a.So(err2, ShouldNotBeNil)

	// Not enough FRMPayloads
	_, err3 := NewBPacket(lorawan.PHYPayload{
		MACPayload: &lorawan.MACPayload{},
	}, Metadata{})
	a.So(err3, ShouldNotBeNil)

	// FRMPayload is not DataPayload
	_, err4 := NewBPacket(lorawan.PHYPayload{
		MACPayload: &lorawan.MACPayload{
			FRMPayload: []lorawan.Payload{
				&lorawan.JoinRequestPayload{},
			},
		},
	}, Metadata{})
	a.So(err4, ShouldNotBeNil)
}

func TestBPacket(t *testing.T) {
	a := New(t)

	payload, _, key := simplePayload()
	input, _ := NewBPacket(payload, Metadata{})

	gOutput := marshalUnmarshal(t, input)

	output := gOutput.(BPacket)

	a.So(output.Payload(), ShouldResemble, payload)
	a.So(output.Metadata(), ShouldResemble, Metadata{})
	outputValidateMIC, _ := output.ValidateMIC(key)
	a.So(outputValidateMIC, ShouldBeTrue)
	a.So(output.Commands(), ShouldBeEmpty)
}

func TestInvalidHPacket(t *testing.T) {
	a := New(t)

	appEUI := newEUI()
	devEUI := newEUI()

	// No MACPayload
	_, err1 := NewHPacket(appEUI, devEUI, lorawan.PHYPayload{}, Metadata{})
	a.So(err1, ShouldNotBeNil)

	// Not a MACPayload
	_, err2 := NewHPacket(appEUI, devEUI, lorawan.PHYPayload{
		MACPayload: &lorawan.JoinRequestPayload{},
	}, Metadata{})
	a.So(err2, ShouldNotBeNil)

	// Not enough FRMPayloads
	_, err3 := NewHPacket(appEUI, devEUI, lorawan.PHYPayload{
		MACPayload: &lorawan.MACPayload{},
	}, Metadata{})
	a.So(err3, ShouldNotBeNil)

	// FRMPayload is not DataPayload
	_, err4 := NewHPacket(appEUI, devEUI, lorawan.PHYPayload{
		MACPayload: &lorawan.MACPayload{
			FRMPayload: []lorawan.Payload{
				&lorawan.JoinRequestPayload{},
			},
		},
	}, Metadata{})
	a.So(err4, ShouldNotBeNil)
}

func TestHPacket(t *testing.T) {
	a := New(t)

	appEUI := newEUI()
	devEUI := newEUI()
	payload, _, key := simplePayload()

	input, _ := NewHPacket(appEUI, devEUI, payload, Metadata{})

	gOutput := marshalUnmarshal(t, input)

	output := gOutput.(HPacket)

	a.So(output.AppEUI().String(), ShouldEqual, appEUI.String())
	a.So(output.DevEUI().String(), ShouldEqual, devEUI.String())
	outPayload, _ := output.Payload(key)
	a.So(string(outPayload), ShouldResemble, "PLD123")
	a.So(output.Metadata(), ShouldResemble, Metadata{})
	a.So(output.FCnt(), ShouldEqual, 1)
}

func TestInvalidAPacket(t *testing.T) {
	a := New(t)

	appEUI := newEUI()
	devEUI := newEUI()

	// No Payload
	_, err1 := NewAPacket(appEUI, devEUI, []byte{}, []Metadata{})
	a.So(err1, ShouldNotBeNil)
}

func TestAPacket(t *testing.T) {
	a := New(t)

	appEUI := newEUI()
	devEUI := newEUI()
	payload := []byte("PLD123")

	input, _ := NewAPacket(appEUI, devEUI, payload, []Metadata{})

	gOutput := marshalUnmarshal(t, input)

	output := gOutput.(APacket)

	a.So(output.Payload(), ShouldResemble, payload)
	a.So(output.DevEUI().String(), ShouldEqual, devEUI.String())
	a.So(output.Payload(), ShouldResemble, payload)
	a.So(output.Metadata(), ShouldBeEmpty)
}

func TestJPacket(t *testing.T) {
	a := New(t)

	appEUI := newEUI()
	devEUI := newEUI()
	devNonce := [2]byte{}
	copy(devEUI[:], randBytes(2))

	input := NewJPacket(appEUI, devEUI, devNonce, Metadata{})

	gOutput := marshalUnmarshal(t, input)

	output := gOutput.(JPacket)

	a.So(output.AppEUI().String(), ShouldEqual, appEUI.String())
	a.So(output.DevEUI().String(), ShouldEqual, devEUI.String())
	a.So(output.DevNonce(), ShouldEqual, devNonce)
	a.So(output.Metadata(), ShouldResemble, Metadata{})
}

func TestInvalidCPacket(t *testing.T) {
	a := New(t)

	appEUI := newEUI()
	devEUI := newEUI()
	nwkSKey := lorawan.AES128Key{}

	// No Payload
	_, err1 := NewCPacket(appEUI, devEUI, []byte{}, nwkSKey)
	a.So(err1, ShouldNotBeNil)
}

func TestCPacket(t *testing.T) {
	a := New(t)

	appEUI := newEUI()
	devEUI := newEUI()
	payload := []byte("PLD123")

	nwkSKey := [16]byte{}
	copy(devEUI[:], randBytes(16))

	input, _ := NewCPacket(appEUI, devEUI, payload, nwkSKey)

	gOutput := marshalUnmarshal(t, input)

	output := gOutput.(CPacket)

	a.So(output.AppEUI().String(), ShouldEqual, appEUI.String())
	a.So(output.DevEUI().String(), ShouldEqual, devEUI.String())
	a.So(output.Payload(), ShouldResemble, payload)
	outputNwkSKey := output.NwkSKey()
	a.So(outputNwkSKey[:], ShouldResemble, nwkSKey[:])
}