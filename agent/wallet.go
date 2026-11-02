package main

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/sha3"
)

func ValidateMoneroAddress(addr string) error {
	addr = strings.TrimSpace(addr)
	if len(addr) != 95 && len(addr) != 106 {
		return fmt.Errorf("unexpected length (%d chars) — standard Monero addresses are 95 characters, integrated addresses 106", len(addr))
	}

	raw, err := decodeMoneroBase58(addr)
	if err != nil {
		return fmt.Errorf("invalid base58 encoding: %w", err)
	}
	if len(raw) < 5 {
		return fmt.Errorf("decoded address is too short")
	}

	payload := raw[:len(raw)-4]
	checksum := raw[len(raw)-4:]

	h := sha3.NewLegacyKeccak256()
	h.Write(payload)
	sum := h.Sum(nil)

	if !bytes.Equal(sum[:4], checksum) {
		return fmt.Errorf("checksum mismatch — the address is likely mistyped, please double-check it")
	}

	switch payload[0] {
	case 18, 19, 42: 
		return nil
	case 24, 25, 36: 
		return nil
	case 53, 54, 63: 
		return nil
	default:
		return fmt.Errorf("unrecognized network byte (0x%x) — this doesn't look like a Monero address", payload[0])
	}
}

const moneroBase58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
const fullBlockSize = 8
const fullEncodedBlockSize = 11

var encodedBlockSizes = [9]int{0, 2, 3, 5, 6, 7, 9, 10, 11}

var moneroBase58Index = func() map[byte]int {
	m := make(map[byte]int, len(moneroBase58Alphabet))
	for i := 0; i < len(moneroBase58Alphabet); i++ {
		m[moneroBase58Alphabet[i]] = i
	}
	return m
}()

func decodeMoneroBase58Block(encoded string, decodedSize int) ([]byte, error) {
	expectedLen := encodedBlockSizes[decodedSize]
	if len(encoded) != expectedLen {
		return nil, fmt.Errorf("unexpected block length")
	}

	num := new(big.Int)
	base := big.NewInt(58)
	digit := new(big.Int)
	for i := 0; i < len(encoded); i++ {
		idx, ok := moneroBase58Index[encoded[i]]
		if !ok {
			return nil, fmt.Errorf("invalid base58 character %q", encoded[i])
		}
		num.Mul(num, base)
		digit.SetInt64(int64(idx))
		num.Add(num, digit)
	}

	rawBytes := num.Bytes()
	if len(rawBytes) > decodedSize {
		return nil, fmt.Errorf("block value overflows its decoded size")
	}
	out := make([]byte, decodedSize)
	copy(out[decodedSize-len(rawBytes):], rawBytes)
	return out, nil
}

func decodeMoneroBase58(addr string) ([]byte, error) {
	fullBlocks := len(addr) / fullEncodedBlockSize
	remainderLen := len(addr) % fullEncodedBlockSize

	remainderDecodedSize := -1
	for decodedSize, encodedLen := range encodedBlockSizes {
		if encodedLen == remainderLen {
			remainderDecodedSize = decodedSize
			break
		}
	}
	if remainderDecodedSize == -1 {
		return nil, fmt.Errorf("invalid address length")
	}

	result := make([]byte, 0, fullBlocks*fullBlockSize+remainderDecodedSize)
	for i := 0; i < fullBlocks; i++ {
		chunk := addr[i*fullEncodedBlockSize : (i+1)*fullEncodedBlockSize]
		b, err := decodeMoneroBase58Block(chunk, fullBlockSize)
		if err != nil {
			return nil, err
		}
		result = append(result, b...)
	}
	if remainderLen > 0 {
		chunk := addr[fullBlocks*fullEncodedBlockSize:]
		b, err := decodeMoneroBase58Block(chunk, remainderDecodedSize)
		if err != nil {
			return nil, err
		}
		result = append(result, b...)
	}
	return result, nil
}
