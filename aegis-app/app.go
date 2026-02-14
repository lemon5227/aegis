package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/tyler-smith/go-bip39"
)

// App struct
type App struct {
	ctx context.Context
}

type Identity struct {
	Mnemonic  string `json:"mnemonic"`
	PublicKey string `json:"publicKey"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) deriveKeypairFromMnemonic(mnemonic string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, nil, errors.New("invalid mnemonic")
	}

	seed := bip39.NewSeed(mnemonic, "")
	hash := sha256.Sum256(seed)
	privateKey := ed25519.NewKeyFromSeed(hash[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return publicKey, privateKey, nil
}

func (a *App) GenerateIdentity() (Identity, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return Identity{}, err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return Identity{}, err
	}

	publicKey, _, err := a.deriveKeypairFromMnemonic(mnemonic)
	if err != nil {
		return Identity{}, err
	}

	return Identity{
		Mnemonic:  mnemonic,
		PublicKey: hex.EncodeToString(publicKey),
	}, nil
}

func (a *App) SignMessage(mnemonic string, message string) (string, error) {
	_, privateKey, err := a.deriveKeypairFromMnemonic(mnemonic)
	if err != nil {
		return "", err
	}

	signature := ed25519.Sign(privateKey, []byte(message))
	return hex.EncodeToString(signature), nil
}

func (a *App) VerifyMessage(publicKeyHex string, message string, signatureHex string) (bool, error) {
	publicKey, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false, err
	}

	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, err
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return false, errors.New("invalid public key length")
	}

	if len(signature) != ed25519.SignatureSize {
		return false, errors.New("invalid signature length")
	}

	return ed25519.Verify(publicKey, []byte(message), signature), nil
}
