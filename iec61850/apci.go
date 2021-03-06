// Copyright 2020 thinkgos (thinkgo@aliyun.com).  All rights reserved.
// Use of this source code is governed by a version 3 of the GNU General
// Public License, license that can be found in the LICENSE file.

package iec61850

import (
	"fmt"
	"log"

	"github.com/themeyic/go-iec61850/asdu"
)

const startFrame byte = 0x68 // 启动字符

// APDU form Max size 255
//      |              APCI                   |       ASDU         |
//      | start | APDU length | control field |       ASDU         |
//                       |          APDU field size(253)           |
// bytes|    1  |    1   |        4           |                    |
const (
	APCICtlFiledSize = 4 // control filed(4)

	APDUSizeMax      = 255                                 // start(1) + length(1) + control field(4) + ASDU
	APDUFieldSizeMax = APCICtlFiledSize + asdu.ASDUSizeMax // control field(4) + ASDU
)

// U帧 控制域功能
const (
	uStartDtActive  byte = 4 << iota // 启动激活 0x04
	uStartDtConfirm                  // 启动确认 0x08
	uStopDtActive                    // 停止激活 0x10
	uStopDtConfirm                   // 停止确认 0x20
	uTestFrActive                    // 测试激活 0x40
	uTestFrConfirm                   // 测试确认 0x80
)

// I帧 含apci和asdu 信息帧.用于编号的信息传输 information
type iAPCI struct {
	sendSN, rcvSN uint16
}

func (sf iAPCI) String() string {
	return fmt.Sprintf("I[sendNO: %d, recvNO: %d]", sf.sendSN, sf.rcvSN)
}

// S帧 只含apci S帧用于主要用确认帧的正确传输,协议称是监视. supervisory
type sAPCI struct {
	rcvSN uint16
}

func (sf sAPCI) String() string {
	return fmt.Sprintf("S[recvNO: %d]", sf.rcvSN)
}

//U帧 只含apci 未编号控制信息 unnumbered
type uAPCI struct {
	function byte // bit8 测试确认
}

func (sf uAPCI) String() string {
	var s string
	switch sf.function {
	case uStartDtActive:
		s = "StartDtActive"
	case uStartDtConfirm:
		s = "StartDtConfirm"
	case uStopDtActive:
		s = "StopDtActive"
	case uStopDtConfirm:
		s = "StopDtConfirm"
	case uTestFrActive:
		s = "TestFrActive"
	case uTestFrConfirm:
		s = "TestFrConfirm"
	default:
		s = "Unknown"
	}
	return fmt.Sprintf("U[function: %s]", s)
}

// newIFrame 创建I帧 ,返回apdu
func newIFrame(sendSN, RcvSN uint16, asdus []byte) ([]byte, error) {
	if len(asdus) > asdu.ASDUSizeMax {
		return nil, fmt.Errorf("ASDU filed large than max %d", asdu.ASDUSizeMax)
	}

	b := make([]byte, len(asdus)+6)

	b[0] = startFrame
	b[1] = byte(len(asdus) + 4)
	b[2] = byte(sendSN << 1)
	b[3] = byte(sendSN >> 7)
	b[4] = byte(RcvSN << 1)
	b[5] = byte(RcvSN >> 7)
	copy(b[6:], asdus)

	return b, nil
}

// newSFrame 创建S帧,返回apdu
func newSFrame(RcvSN uint16) []byte {
	return []byte{startFrame, 4, 0x01, 0x00, byte(RcvSN << 1), byte(RcvSN >> 7)}
}



type GooseDOApp struct {
	GocbRef            string    `asn1:"tag:0"`
	TimeToLive         int       `asn1:"tag:1"`
	DataSet            string    `asn1:"tag:2"`
	GoID               string    `asn1:"optional,tag:3"`
	UtcTime            [8]byte   `asn1:"tag:4"`
	StNum              int       `asn1:"tag:5"`
	SqNum              int       `asn1:"tag:6"`
	Test               bool      `asn1:"tag:7"`
	ConfRev            int       `asn1:"tag:8"`
	NeedsCommissioning bool      `asn1:"tag:9"`
	NumDataSetEntries  int       `asn1:"tag:10"`
	AllData            AllDataDO `asn1:"tag:11"`
}

type AllDataDO struct {
	DO0 bool `asn1:"tag:3"`
	DO1 bool `asn1:"tag:3"`
	DO2 bool `asn1:"tag:3"`
	DO3 bool `asn1:"tag:3"`
	DO4 bool `asn1:"tag:3"`
	DO5 bool `asn1:"tag:3"`
	DO6 bool `asn1:"tag:3"`
	DO7 bool `asn1:"tag:3"`
}


const (
	GOOSE_TYPE_ID uint32 = 35000 //[2]byte{0x88, 0xb8}
	SV_TYPE_ID    uint32 = 35002 //[2]byte{0x88, 0xba}
)

// 以太网报文头
type EtherHeader struct {
	DstHwaddr []byte
	LocHwaddr []byte
	VlanTag   []byte
	TypeId    uint32
	AppId     uint32
}

func PackEtherPacket(header EtherHeader, apdu []byte) []byte {
	if header.TypeId != GOOSE_TYPE_ID && header.TypeId != SV_TYPE_ID {
		log.Println("PackEtherPacket: not goose or sv packet.")
		return nil
	}

	packet := make([]byte, 26+len(apdu))
	idx := 0
	copy(packet[idx:], header.DstHwaddr)
	idx += 6
	copy(packet[idx:], header.LocHwaddr)
	idx += 6
	if len(header.VlanTag) != 0 {
		copy(packet[idx:], header.VlanTag)
		idx += 4
	}
	EncodeUint(header.TypeId, packet[idx:idx+2])
	idx += 2
	EncodeUint(header.AppId, packet[idx:idx+2])
	idx += 2
	EncodeUint(uint32(len(apdu)+8), packet[idx:idx+2])
	idx += 2
	EncodeUint(0, packet[idx:idx+4])
	idx += 4
	copy(packet[idx:], apdu)
	idx += len(apdu)

	return packet[:idx]
}

func EncodeUint(iVal uint32, b []byte) {
	b_len := len(b)
	if b_len > 4 {
		b_len = 4
	}
	for i := 0; i < b_len; i++ {
		shift := 8 * uint(b_len-i-1)
		mask := uint32(0xff) << shift
		b[i] = byte((iVal & mask) >> shift)
	}
}









// newUFrame 创建U帧,返回apdu
func newInitFrame() []byte {
	//return []byte{startFrame, 4, which | 0x03, 0x00, 0x00, 0x00}
	startMark = 0
	endMark = 20
	return []byte{0x03,0x00,0x00,0x16,0x11,0xe0,0x00,0x00,0x00,0x01,0x00,0xc0,0x01,0x0a,0xc2,0x02,0x00,0x01,0xc1,0x02,0x00,0x01}

	//return []byte{0xa8,0x26,0x80,0x03,0x00,0xfd,0xe8,0x81,0x01,0x05,0x82,0x01,0x05,0x83,0x01,0x0a,0xa4,0x16,0x80,0x01,0x01,0x81,0x03,0x05,0xf1,0x00,0x82,0x0c,0x03,0xee,0x1c,0x00,0x00,0x04,0x08,0x00,0x00,0x79,0xef,0x18}
}

func newSecondUFrame() []byte {
	//return []byte{startFrame, 4, which | 0x03, 0x00, 0x00, 0x00}
	//return []byte{0x03,0x00,0x00,0x16,0x11,0xe0,0x00,0x00,0x00,0x01,0x00,0xc0,0x01,0x0a,0xc2,0x02,0x00,0x01,0xc1,0x02,0x00,0x01}
	startMark = 0
	endMark = 20
	return   []byte{0x03,0x00,0x00,0xba,0x02,0xf0,0x80,0x0d,0xb1,0x05,0x06,0x13,0x01,0x00,0x16,0x01,0x02,0x14,0x02,0x00,0x02,0x33,0x02,0x00,0x01,0x34,0x02,0x00,0x01,0xc1,0x9b,0x31,0x81,0x98,0xa0,0x03,0x80,0x01,0x01,0xa2,0x81,0x90,0x81,0x04,0x00,0x00,0x00,0x01,0x82,0x04,0x00,0x00,0x00,0x01,0xa4,0x23,0x30,0x0f,0x02,0x01,0x01,0x06,0x04,0x52,0x01,0x00,0x01,0x30,0x04,0x06,0x02,0x51,0x01,0x30,0x10,0x02,0x01,0x03,0x06,0x05,0x28,0xca,0x22,0x02,0x01,0x30,0x04,0x06,0x02,0x51,0x01,0x61,0x5d,0x30,0x5b,0x02,0x01,0x01,0xa0,0x56,0x60,0x54,0xa1,0x07,0x06,0x05,0x28,0xca,0x22,0x02,0x03,0xa2,0x06,0x06,0x04,0x2b,0xce,0x0f,0x0d,0xa3,0x03,0x02,0x01,0x0c,0xa6,0x06,0x06,0x04,0x2b,0xce,0x0f,0x0d,0xa7,0x03,0x02,0x01,0x01,0xbe,0x2f,0x28,0x2d,0x02,0x01,0x03,0xa0,0x28,0xa8,0x26,0x80,0x03,0x00,0xfd,0xe8,0x81,0x01,0x05,0x82,0x01,0x05,0x83,0x01,0x0a,0xa4,0x16,0x80,0x01,0x01,0x81,0x03,0x05,0xf1,0x00,0x82,0x0c,0x03,0xee,0x1c,0x00,0x00,0x04,0x08,0x00,0x00,0x79,0xef,0x18}



	//return []byte{0xa8,0x26,0x80,0x03,0x00,0xfd,0xe8,0x81,0x01,0x05,0x82,0x01,0x05,0x83,0x01,0x0a,0xa4,0x16,0x80,0x01,0x01,0x81,0x03,0x05,0xf1,0x00,0x82,0x0c,0x03,0xee,0x1c,0x00,0x00,0x04,0x08,0x00,0x00,0x79,0xef,0x18}
}



func newThreeUFrame(which byte) []byte {
	//return []byte{startFrame, 4, which | 0x03, 0x00, 0x00, 0x00}
	//return []byte{0x03,0x00,0x00,0x16,0x11,0xe0,0x00,0x00,0x00,0x01,0x00,0xc0,0x01,0x0a,0xc2,0x02,0x00,0x01,0xc1,0x02,0x00,0x01}
	startMark = 0
	endMark = 47
	return   []byte{0x03,0x00,0x00,0x24,0x02,0xf0,0x80,0x01,0x00,0x01,0x00,0x61,0x17,0x30,0x15,0x02,0x01,0x03,0xa0,0x10,0xa0,0x0e,0x02,0x01,0x01,0xa1,0x09,0xa0,0x03,0x80,0x01,0x09,0xa1,0x02,0x80,0x00}


	//return []byte{0xa8,0x26,0x80,0x03,0x00,0xfd,0xe8,0x81,0x01,0x05,0x82,0x01,0x05,0x83,0x01,0x0a,0xa4,0x16,0x80,0x01,0x01,0x81,0x03,0x05,0xf1,0x00,0x82,0x0c,0x03,0xee,0x1c,0x00,0x00,0x04,0x08,0x00,0x00,0x79,0xef,0x18}
}

//func newFourUFrame(which byte) []byte {
////
////	//return []byte{startFrame, 4, which | 0x03, 0x00, 0x00, 0x00}
////	//return []byte{0x03,0x00,0x00,0x16,0x11,0xe0,0x00,0x00,0x00,0x01,0x00,0xc0,0x01,0x0a,0xc2,0x02,0x00,0x01,0xc1,0x02,0x00,0x01}
////	test11 = 0
////	test22 = 3000
////	return   []byte{0x03,0x00,0x00,0x2d,0x02,0xf0,0x80,0x01,0x00,0x01,0x00,0x61,0x20,0x30,0x1e,0x02,0x01,0x03,0xa0,0x19,0xa0,0x17,0x02,0x01,0x02,0xa1,0x12,0xa0,0x03,0x80,0x01,0x00,0xa1,0x0b,0x81,0x09,0x50,0x52,0x53,0x37,0x37,0x38,0x52,0x43,0x44}
////
////
////
////	//return []byte{0xa8,0x26,0x80,0x03,0x00,0xfd,0xe8,0x81,0x01,0x05,0x82,0x01,0x05,0x83,0x01,0x0a,0xa4,0x16,0x80,0x01,0x01,0x81,0x03,0x05,0xf1,0x00,0x82,0x0c,0x03,0xee,0x1c,0x00,0x00,0x04,0x08,0x00,0x00,0x79,0xef,0x18}
////}

func newFineUFrame() []byte {

	//return []byte{startFrame, 4, which | 0x03, 0x00, 0x00, 0x00}
	//return []byte{0x03,0x00,0x00,0x16,0x11,0xe0,0x00,0x00,0x00,0x01,0x00,0xc0,0x01,0x0a,0xc2,0x02,0x00,0x01,0xc1,0x02,0x00,0x01}
	startMark = 0
	endMark = 40
	return   []byte{0x03,0x00,0x00,0x39,0x02,0xf0,0x80,0x01,0x00,0x01,0x00,0x61,0x2c,0x30,0x2a,0x02,0x01,0x03,0xa0,0x25,0xa0,0x23,0x02,0x01,0x11,0xa4,0x1e,0xa1,0x1c,0xa0,0x1a,0x30,0x18,0xa0,0x16,0xa1,0x14,0x1a,0x09,0x50,0x52,0x53,0x37,0x37,0x38,0x52,0x43,0x44,0x1a,0x07,0x4c,0x4c,0x4e,0x30,0x24,0x43,0x46}



		//return []byte{0xa8,0x26,0x80,0x03,0x00,0xfd,0xe8,0x81,0x01,0x05,0x82,0x01,0x05,0x83,0x01,0x0a,0xa4,0x16,0x80,0x01,0x01,0x81,0x03,0x05,0xf1,0x00,0x82,0x0c,0x03,0xee,0x1c,0x00,0x00,0x04,0x08,0x00,0x00,0x79,0xef,0x18}
}






// APCI apci 应用规约控制信息
type APCI struct {
	start                  byte
	apduFiledLen           byte // control + asdu 的长度
	ctr1, ctr2, ctr3, ctr4 byte
}

// return frame type , APCI, remain data
func parse(apdu []byte) (interface{}, []byte) {
	apci := APCI{apdu[0], apdu[1], apdu[2], apdu[3], apdu[4], apdu[5]}
	if apci.ctr1&0x01 == 0 {
		return iAPCI{
			sendSN: uint16(apci.ctr1)>>1 + uint16(apci.ctr2)<<7,
			rcvSN:  uint16(apci.ctr3)>>1 + uint16(apci.ctr4)<<7,
		}, apdu[6:]
	}
	if apci.ctr1&0x03 == 0x01 {
		return sAPCI{
			rcvSN: uint16(apci.ctr3)>>1 + uint16(apci.ctr4)<<7,
		}, apdu[6:]
	}
	// apci.ctrl&0x03 == 0x03
	return uAPCI{
		function: apci.ctr1 & 0xfc,
	}, apdu[6:]
}
