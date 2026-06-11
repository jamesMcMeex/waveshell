// Package scanner walks configured library paths, reads audio file metadata,
// and emits scan progress/result Msgs for database insertion.
package scanner

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type streamInfo struct {
	SampleRate int
	BitDepth   int
	Bitrate    int
	Channels   int
	DurationMs int
}

func inspectStream(f *os.File, ext string) (*streamInfo, error) {
	switch ext {
	case ".flac":
		return inspectFLAC(f)
	case ".mp3":
		return inspectMP3(f)
	case ".m4a", ".alac":
		return inspectMP4(f)
	case ".aiff", ".aif":
		return inspectAIFF(f)
	case ".wav":
		return inspectWAV(f)
	case ".ogg":
		return inspectOGG(f)
	}
	return nil, fmt.Errorf("unsupported format: %s", ext)
}

func inspectFLAC(f *os.File) (*streamInfo, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var header [4]byte
	if _, err := io.ReadFull(f, header[:]); err != nil {
		return nil, fmt.Errorf("read fLaC marker: %w", err)
	}
	if string(header[:]) != "fLaC" {
		return nil, fmt.Errorf("not a FLAC file")
	}

	var metaBlockHeader [4]byte
	if _, err := io.ReadFull(f, metaBlockHeader[:]); err != nil {
		return nil, fmt.Errorf("read meta block header: %w", err)
	}

	blockType := metaBlockHeader[0] & 0x7f
	if blockType != 0 {
		return nil, fmt.Errorf("expected STREAMINFO block (type 0), got %d", blockType)
	}

	blockLen := int(metaBlockHeader[1])<<16 | int(metaBlockHeader[2])<<8 | int(metaBlockHeader[3])
	if blockLen < 34 {
		return nil, fmt.Errorf("STREAMINFO block too short: %d", blockLen)
	}

	var streamInfoBuf [34]byte
	if _, err := io.ReadFull(f, streamInfoBuf[:]); err != nil {
		return nil, fmt.Errorf("read STREAMINFO: %w", err)
	}

	minBlockSize := int(binary.BigEndian.Uint16(streamInfoBuf[0:2]))
	_ = minBlockSize

	sampleRate := int(binary.BigEndian.Uint32(streamInfoBuf[10:14]) >> 12)
	channels := int((streamInfoBuf[12]>>1)&0x07) + 1
	bitDepth := int((binary.BigEndian.Uint16(streamInfoBuf[12:14])>>5)&0x1F) + 1

	totalSamples := int64(streamInfoBuf[13]&0x0F)<<32 |
		int64(streamInfoBuf[14])<<24 |
		int64(streamInfoBuf[15])<<16 |
		int64(streamInfoBuf[16])<<8 |
		int64(streamInfoBuf[17])

	var durationMs int
	if sampleRate > 0 && totalSamples > 0 {
		durationMs = int(totalSamples * 1000 / int64(sampleRate))
	}

	return &streamInfo{
		SampleRate: sampleRate,
		BitDepth:   bitDepth,
		Channels:   channels,
		DurationMs: durationMs,
	}, nil
}

func inspectMP3(f *os.File) (*streamInfo, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	skipID3v2(f)

	buf := make([]byte, 4)
	for {
		_, err := io.ReadFull(f, buf)
		if err != nil {
			return nil, fmt.Errorf("finding sync: %w", err)
		}

		if buf[0] == 0xFF && (buf[1]&0xE0) == 0xE0 {
			header := uint32(buf[0])<<24 | uint32(buf[1])<<16 |
				uint32(buf[2])<<8 | uint32(buf[3])

			version := (header >> 19) & 0x3
			layer := (header >> 17) & 0x3
			bitrateIndex := (header >> 12) & 0xF
			sampleRateIndex := (header >> 10) & 0x3

			if version == 1 || layer == 0 || bitrateIndex == 0 || bitrateIndex == 0xF {
				continue
			}

			bitrates := [2][3][16]int{
				{
					{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448, 0},
					{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 0},
					{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0},
				},
				{
					{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256, 0},
					{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
					{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
				},
			}

			sampleRates := [2][4]int{
				{44100, 48000, 32000, 0},
				{22050, 24000, 16000, 0},
			}

			var vIdx int
			switch version {
			case 3:
				vIdx = 0
			case 2, 0:
				vIdx = 1
			default:
				continue
			}

			lIdx := 3 - layer
			if lIdx > 2 {
				lIdx = 2
			}

			srIdx := sampleRateIndex
			if srIdx > 2 {
				srIdx = 2
			}

			bitrate := bitrates[vIdx][lIdx][bitrateIndex]
			sampleRate := sampleRates[vIdx][srIdx]

			if bitrate == 0 || sampleRate == 0 {
				continue
			}

			info, err := f.Stat()
			if err != nil {
				return &streamInfo{SampleRate: sampleRate, Bitrate: bitrate}, nil
			}

			durationMs := int((info.Size() * 8) / int64(bitrate) / 1000)

			return &streamInfo{
				SampleRate: sampleRate,
				Bitrate:    bitrate,
				DurationMs: durationMs,
			}, nil
		}

		if _, err := f.Seek(-3, io.SeekCurrent); err != nil {
			return nil, err
		}
	}
}

func inspectMP4(f *os.File) (*streamInfo, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	buf := make([]byte, 8)
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, fmt.Errorf("read ftyp: %w", err)
	}

	ftypSize := int64(binary.BigEndian.Uint32(buf[0:4]))
	if ftypSize < 8 {
		return nil, fmt.Errorf("invalid ftyp size")
	}
	if _, err := f.Seek(ftypSize-8, io.SeekCurrent); err != nil {
		return nil, err
	}

	fileSize, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var pos int64
	for pos = ftypSize; pos < fileSize-8; {
		if _, err := f.Seek(pos, io.SeekStart); err != nil {
			return nil, err
		}
		if _, err := io.ReadFull(f, buf); err != nil {
			return nil, err
		}

		atomSize := int64(binary.BigEndian.Uint32(buf[0:4]))
		atomType := string(buf[4:8])

		if atomSize == 0 {
			atomSize = fileSize - pos
		}
		if atomSize < 8 {
			break
		}

		if atomType == "moov" {
			return parseMoovAtom(f, pos)
		}

		pos += atomSize
	}

	return nil, fmt.Errorf("moov atom not found")
}

func parseMoovAtom(f *os.File, moovPos int64) (*streamInfo, error) {
	atomSizeBytes := readBytesAt(f, moovPos, 4)
	if atomSizeBytes == nil {
		return nil, fmt.Errorf("cannot read moov atom size")
	}
	moovEnd := moovPos + int64(binary.BigEndian.Uint32(atomSizeBytes))

	for pos := moovPos + 8; pos < moovEnd-8; {
		atomHeader := readBytesAt(f, pos, 8)
		if atomHeader == nil {
			break
		}
		atomSize := int64(binary.BigEndian.Uint32(atomHeader[0:4]))
		atomType := string(atomHeader[4:8])
		if atomSize < 8 {
			break
		}

		if atomType == "mvhd" {
			return parseMvhdAtom(f, pos)
		}

		pos += atomSize
	}

	return nil, fmt.Errorf("mvhd atom not found")
}

func parseMvhdAtom(f *os.File, mvhdPos int64) (*streamInfo, error) {
	data := readBytesAt(f, mvhdPos+8, 80)
	if data == nil {
		return nil, fmt.Errorf("cannot read mvhd")
	}

	version := data[0]
	var timeScale int
	var duration int

	if version == 1 {
		timeScale = int(binary.BigEndian.Uint32(data[20:24]))
		durationVal := int64(binary.BigEndian.Uint64(data[24:32]))
		if timeScale > 0 {
			duration = int(durationVal * 1000 / int64(timeScale))
		}
	} else {
		timeScale = int(binary.BigEndian.Uint32(data[12:16]))
		durationVal := int64(binary.BigEndian.Uint32(data[16:20]))
		if timeScale > 0 {
			duration = int(durationVal * 1000 / int64(timeScale))
		}
	}

	return &streamInfo{
		SampleRate: 44100,
		DurationMs: duration,
	}, nil
}

func readBytesAt(f *os.File, offset int64, length int) []byte {
	buf := make([]byte, length)
	_, err := f.ReadAt(buf, offset)
	if err != nil {
		return nil
	}
	return buf
}

func inspectAIFF(f *os.File) (*streamInfo, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var formID [4]byte
	var chunkSize int32
	var fileType [4]byte

	if err := readExact(f, formID[:]); err != nil {
		return nil, fmt.Errorf("read FORM: %w", err)
	}
	if string(formID[:]) != "FORM" {
		return nil, fmt.Errorf("not AIFF: missing FORM")
	}
	if err := binary.Read(f, binary.BigEndian, &chunkSize); err != nil {
		return nil, fmt.Errorf("read size: %w", err)
	}
	if err := readExact(f, fileType[:]); err != nil {
		return nil, fmt.Errorf("read file type: %w", err)
	}

	remaining := int64(chunkSize) - 4
	for remaining > 8 {
		var ckID [4]byte
		var ckSize int32
		if err := readExact(f, ckID[:]); err != nil {
			break
		}
		if err := binary.Read(f, binary.BigEndian, &ckSize); err != nil {
			break
		}

		padSize := ckSize
		if padSize%2 != 0 {
			padSize++
		}

		if string(ckID[:]) == "COMM" {
			var channels int16
			var sampleFrames int32
			var bitDepth int16
			if err := binary.Read(f, binary.BigEndian, &channels); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.BigEndian, &sampleFrames); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.BigEndian, &bitDepth); err != nil {
				return nil, err
			}

			var sampleRateBuf [10]byte
			if err := readExact(f, sampleRateBuf[:]); err != nil {
				return nil, err
			}

			sampleRate := parseExtended80(sampleRateBuf[:])
			duration := 0
			if sampleRate > 0 {
				duration = int(int64(sampleFrames) * 1000 / int64(sampleRate))
			}

			return &streamInfo{
				SampleRate: sampleRate,
				BitDepth:   int(bitDepth),
				Channels:   int(channels),
				DurationMs: duration,
			}, nil
		}

		if _, err := f.Seek(int64(padSize), io.SeekCurrent); err != nil {
			break
		}
		remaining -= int64(padSize) + 8
	}

	return nil, fmt.Errorf("COMM chunk not found")
}

func inspectWAV(f *os.File) (*streamInfo, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var riffID [4]byte
	var fileSize int32
	var waveID [4]byte

	if err := readExact(f, riffID[:]); err != nil {
		return nil, fmt.Errorf("read RIFF: %w", err)
	}
	if string(riffID[:]) != "RIFF" {
		return nil, fmt.Errorf("not WAV: missing RIFF")
	}
	if err := binary.Read(f, binary.LittleEndian, &fileSize); err != nil {
		return nil, fmt.Errorf("read size: %w", err)
	}
	if err := readExact(f, waveID[:]); err != nil {
		return nil, fmt.Errorf("read WAVE: %w", err)
	}
	if string(waveID[:]) != "WAVE" {
		return nil, fmt.Errorf("not WAV: missing WAVE")
	}

	remaining := int64(fileSize) - 4
	for remaining > 8 {
		var ckID [4]byte
		var ckSize int32
		if err := readExact(f, ckID[:]); err != nil {
			break
		}
		if err := binary.Read(f, binary.LittleEndian, &ckSize); err != nil {
			break
		}

		if string(ckID[:]) == "fmt " {
			var audioFormat, channels int16
			var sampleRate, byteRate int32
			var blockAlign, bitDepth int16

			if err := binary.Read(f, binary.LittleEndian, &audioFormat); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.LittleEndian, &channels); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.LittleEndian, &sampleRate); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.LittleEndian, &byteRate); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.LittleEndian, &blockAlign); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.LittleEndian, &bitDepth); err != nil {
				return nil, err
			}

			return &streamInfo{
				SampleRate: int(sampleRate),
				BitDepth:   int(bitDepth),
				Channels:   int(channels),
				Bitrate:    int(byteRate * 8 / 1000),
			}, nil
		}

		padSize := ckSize
		if padSize%2 != 0 {
			padSize++
		}
		if _, err := f.Seek(int64(padSize), io.SeekCurrent); err != nil {
			break
		}
		remaining -= int64(padSize) + 8
	}

	return nil, fmt.Errorf("fmt chunk not found")
}

func inspectOGG(f *os.File) (*streamInfo, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var capture [4]byte
	if err := readExact(f, capture[:]); err != nil {
		return nil, fmt.Errorf("read OggS: %w", err)
	}
	if string(capture[:]) != "OggS" {
		return nil, fmt.Errorf("not OGG")
	}

	if _, err := f.Seek(22, io.SeekCurrent); err != nil {
		return nil, err
	}

	var numSegments byte
	if err := binary.Read(f, binary.LittleEndian, &numSegments); err != nil {
		return nil, err
	}

	segTable := make([]byte, numSegments)
	if err := readExact(f, segTable); err != nil {
		return nil, err
	}

	var packetSize int
	for _, s := range segTable {
		packetSize += int(s)
	}

	vorbisHeader := make([]byte, packetSize)
	if err := readExact(f, vorbisHeader); err != nil {
		return nil, err
	}

	if len(vorbisHeader) < 30 || string(vorbisHeader[0:7]) != "\x01vorbis" {
		return nil, fmt.Errorf("not Vorbis")
	}

	sampleRate := int(binary.LittleEndian.Uint32(vorbisHeader[12:16]))
	channels := int(vorbisHeader[11])

	return &streamInfo{
		SampleRate: sampleRate,
		Channels:   channels,
	}, nil
}

func skipID3v2(f *os.File) {
	var buf [10]byte
	if _, err := io.ReadFull(f, buf[:]); err != nil {
		_, _ = f.Seek(0, io.SeekStart)
		return
	}
	if string(buf[0:3]) != "ID3" {
		_, _ = f.Seek(0, io.SeekStart)
		return
	}
	size := int(buf[6])<<21 | int(buf[7])<<14 | int(buf[8])<<7 | int(buf[9])
	_, _ = f.Seek(int64(size), io.SeekCurrent)
}

func readExact(r io.Reader, buf []byte) error {
	_, err := io.ReadFull(r, buf)
	return err
}

func parseExtended80(buf []byte) int {
	if len(buf) < 10 {
		return 44100
	}
	expField := binary.BigEndian.Uint16(buf[0:2])
	if expField&0x7FFF == 0 {
		return 0
	}
	exp := int(expField&0x7FFF) - 16383
	var mantissa uint64
	for i := 0; i < 8; i++ {
		mantissa = (mantissa << 8) | uint64(buf[2+i])
	}
	if exp < 0 {
		return int(mantissa >> uint(63-exp))
	}
	if exp > 63 {
		return 0
	}
	intPart := int(mantissa >> uint(63-exp))
	roundBit := uint64(1) << uint(62-exp)
	if (mantissa & roundBit) != 0 {
		intPart++
	}
	return intPart
}
