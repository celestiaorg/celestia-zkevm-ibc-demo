package groth16

import (
	"bytes"
	"fmt"
	"os"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
)

func SerializeVerifyingKey(vk groth16.VerifyingKey) ([]byte, error) {
	var buf bytes.Buffer
	_, err := vk.WriteTo(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DeserializeVerifyingKey(vkProto []byte) (groth16.VerifyingKey, error) {
	// vk := groth16.NewVerifyingKey(ecc.BN254)
	// _, err := vk.ReadFrom(bytes.NewReader(vkProto))
	// if err != nil {
	// 	return nil, err
	// }
	dir, err := os.Getwd()
	vkFile, err := os.Open(dir + "/ibc/lightclients/groth16/groth16_vk.bin")
	if err != nil {
		return nil, fmt.Errorf("failed to open vk file %w", err)
	}
	vk := groth16.NewVerifyingKey(ecc.BN254)
	_, err = vk.ReadFrom(vkFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read vk file %w", err)
	}
	return vk, nil
}
