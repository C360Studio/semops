package mavlink

var crcExtraByMessageID = map[uint32]uint8{
	MessageIDHeartbeat:         50,
	MessageIDAttitude:          39,
	MessageIDGlobalPositionInt: 104,
	MessageIDCommandLong:       152,
	MessageIDCommandAck:        143,
	MessageIDBatteryStatus:     154,
}

func calculateChecksum(dataWithoutSTX []byte, messageID uint32) uint16 {
	crcExtra, hasCRCExtra := standardCRCExtra(messageID)
	return calculateChecksumWithExtra(dataWithoutSTX, crcExtra, hasCRCExtra)
}

func calculateChecksumWithExtra(dataWithoutSTX []byte, crcExtra uint8, hasCRCExtra bool) uint16 {
	crc := uint16(0xffff)
	for _, b := range dataWithoutSTX {
		crc = crcAccumulate(b, crc)
	}
	if hasCRCExtra {
		crc = crcAccumulate(crcExtra, crc)
	}
	return crc
}

func standardCRCExtra(messageID uint32) (uint8, bool) {
	crcExtra, ok := crcExtraByMessageID[messageID]
	return crcExtra, ok
}

func crcAccumulate(b byte, crc uint16) uint16 {
	tmp := b ^ uint8(crc&0xff)
	tmp ^= tmp << 4
	return (crc >> 8) ^ (uint16(tmp) << 8) ^ (uint16(tmp) << 3) ^ (uint16(tmp) >> 4)
}
