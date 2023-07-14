package csp

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
)

type SignedPid struct {
	Pid       uint32
	Signature []byte
}

const minSize = 5

func (s *SignedPid) FromBytes(b []byte) error {
	if len(b) < minSize {
		return fmt.Errorf("signed pid must have at least %d bytes, has %d", minSize, len(b))
	}
	s.Pid = binary.LittleEndian.Uint32(b[:4])
	s.Signature = b[4:]
	return nil
}

func (s *SignedPid) ToBytes() []byte {
	pid := make([]byte, 4)
	binary.LittleEndian.PutUint32(pid, s.Pid)
	return append(pid, s.Signature...)
}

func hashUint32(u uint32) []byte {
	uAsData := make([]byte, 4)
	binary.LittleEndian.PutUint32(uAsData, u)
	uHash := sha256.New()
	uHash.Write(uAsData)
	return uHash.Sum(nil)
}

func SignPid(pid uint32, key crypto.PrivateKey) *SignedPid {
	pidSum := hashUint32(pid)
	prv, _ := key.(*rsa.PrivateKey)
	signature, _ := rsa.SignPKCS1v15(rand.Reader, prv, crypto.SHA256, pidSum)

	return &SignedPid{
		Pid:       pid,
		Signature: signature,
	}
}

func (s *SignedPid) VerifyPid(key crypto.PublicKey) error {
	pub, _ := key.(*rsa.PublicKey)
	pidSum := hashUint32(s.Pid)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, pidSum, s.Signature)
}

// From https://stackoverflow.com/a/67035019/7557549, thanks Ninazu <3
func EncryptOAEPChunks(hash hash.Hash, random io.Reader, public *rsa.PublicKey, msg []byte, label []byte) ([]byte, error) {
	msgLen := len(msg)
	step := public.Size() - 2*hash.Size() - 2
	var encryptedBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		encryptedBlockBytes, err := rsa.EncryptOAEP(hash, random, public, msg[start:finish], label)
		if err != nil {
			return nil, err
		}

		encryptedBytes = append(encryptedBytes, encryptedBlockBytes...)
	}

	return encryptedBytes, nil
}

func DecryptOAEPChunks(hash hash.Hash, random io.Reader, private *rsa.PrivateKey, msg []byte, label []byte) ([]byte, error) {
	msgLen := len(msg)
	step := private.PublicKey.Size()
	var decryptedBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		decryptedBlockBytes, err := rsa.DecryptOAEP(hash, random, private, msg[start:finish], label)
		if err != nil {
			return nil, err
		}

		decryptedBytes = append(decryptedBytes, decryptedBlockBytes...)
	}

	return decryptedBytes, nil
}
