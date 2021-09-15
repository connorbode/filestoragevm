// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package filestoragevm

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/json"
)

var (
	errBadData     = errors.New("data must be base 58 repr. of 32 bytes")
	errNoSuchBlock = errors.New("couldn't get block from database. Does it exist?")
)

// Service is the API service for this VM
type Service struct{ vm *VM }

// ProposeBlockArgs are the arguments to function ProposeValue
type ProposeBlockArgs struct {
	// Data in the block. Must be base 58 encoding of 32 bytes.
	Data string `json:"data"`
}

// ProposeBlockReply is the reply from function ProposeBlock
type ProposeBlockReply struct{ Success bool }

// ProposeBlock is an API method to propose a new block whose data is [args].Data.
// [args].Data must be a string repr. of a 32 byte array
func (s *Service) ProposeBlock(_ *http.Request, args *ProposeBlockArgs, reply *ProposeBlockReply) error {
	bytes, err := formatting.Decode(formatting.CB58, args.Data)
	if err != nil || len(bytes) != dataLen {
		return errBadData
	}
	var data [dataLen]byte         // The data as an array of bytes
	copy(data[:], bytes[:dataLen]) // Copy the bytes in dataSlice to data
	s.vm.proposeBlock(data)
	reply.Success = true
	return nil
}

// APIBlock is the API representation of a block
type APIBlock struct {
	Timestamp json.Uint64 `json:"timestamp"` // Timestamp of most recent block
	Data      string      `json:"data"`      // Data in the most recent block. Base 58 repr. of 5 bytes.
	ID        string      `json:"id"`        // String repr. of ID of the most recent block
	ParentID  string      `json:"parentID"`  // String repr. of ID of the most recent block's parent
}

// GetBlockArgs are the arguments to GetBlock
type GetBlockArgs struct {
	// ID of the block we're getting.
	// If left blank, gets the latest block
	ID string
}

// GetBlockReply is the reply from GetBlock
type GetBlockReply struct {
	APIBlock
}

// GetBlock gets the block whose ID is [args.ID]
// If [args.ID] is empty, get the latest block
func (s *Service) GetBlock(_ *http.Request, args *GetBlockArgs, reply *GetBlockReply) error {
	// If an ID is given, parse its string representation to an ids.ID
	// If no ID is given, ID becomes the ID of last accepted block
	var id ids.ID
	var err error
	if args.ID == "" {
		id, err = s.vm.LastAccepted()
		if err != nil {
			return fmt.Errorf("problem finding the last accepted ID: %s", err)
		}
	} else {
		id, err = ids.FromString(args.ID)
		if err != nil {
			return errors.New("problem parsing ID")
		}
	}

	// Get the block from the database
	blockInterface, err := s.vm.GetBlock(id)
	if err != nil {
		return errNoSuchBlock
	}

	block, ok := blockInterface.(*Block)
	if !ok { // Should never happen but better to check than to panic
		return errBadData
	}

	// Fill out the response with the block's data
	reply.APIBlock.ID = block.ID().String()
	reply.APIBlock.Timestamp = json.Uint64(block.Timestamp().Unix())
	reply.APIBlock.ParentID = block.Parent().String()
	reply.Data, err = formatting.EncodeWithChecksum(formatting.CB58, block.Data[:])

	return err
}

type CreateAddressArgs struct {
}

type CreateAddressReply struct {
	PrivateKey string `json:"privateKey"`
	PublicKey string `json:"publicKey"`
}

func (s *Service) CreateAddress(_ *http.Request, args *CreateAddressArgs, reply *CreateAddressReply) error {
	var err error
	factory := crypto.FactorySECP256K1R{}
	skIntf, err := factory.NewPrivateKey()
	if err != nil {
		return fmt.Errorf("problem generating private key: %w", err)
	}
	sk := skIntf.(*crypto.PrivateKeySECP256K1R)
	privKeyStr, _ := formatting.EncodeWithChecksum(formatting.CB58, sk.Bytes())
	pubKeyStr, _ := formatting.EncodeWithChecksum(formatting.CB58, sk.PublicKey().Bytes())
	reply.PublicKey = pubKeyStr
	reply.PrivateKey = privKeyStr
	return err
}

type GetBlockHeightArgs struct {
}

type GetBlockHeightReply struct {
	BlockHeight string `json:"blockHeight"`
}

func (s *Service) GetBlockHeight(_ *http.Request, args *GetBlockHeightArgs, reply *GetBlockHeightReply) error {
	var err error
	var id ids.ID
	id, err = s.vm.State.GetLastAccepted(s.vm.DB)
	reply.BlockHeight = id.String()
	return err
}

type GetStorageCostArgs struct {
}

type GetStorageCostReply struct {
	Cost string `json:"cost"`
}

func (s *Service) GetStorageCost(_ *http.Request, args *GetStorageCostArgs, reply *GetStorageCostReply) error {
	var err error
	reply.Cost = "0" // TODO
	return err
}

type GetBalanceArgs struct {
	PublicKey string
}

type GetBalanceReply struct {
	Balance string `json:"balance"`
}

func (s *Service) GetBalance(_ *http.Request, args *GetBalanceArgs, reply *GetBalanceReply) error {
	var err error
	reply.Balance = "0" // TODO
	return err
}

type VerifySignatureArgs struct {
	PublicKey string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
	Signature string `json:"signature"`
	Message string `json:"message"`
}

type VerifySignatureReply struct {
	Valid bool `json:"valid"`
	/*
	Expected string `json:"expected"`
	ExpectedChecksum string `json:"expectedChecksum"`
	Actual string `json:"actual"`
	ExpectedValid bool `json:"expectedValid"`
	ExpectedPubkey string `json:"expectedPubkey"`
	ActualPubkey string `json:"actualPubkey"`
	*/
}

func (s *Service) VerifySignature(_ *http.Request, args *VerifySignatureArgs, reply *VerifySignatureReply) error {
	var err error
	decodedPubkey, _ := formatting.Decode(formatting.CB58, args.PublicKey)
	decodedSigBytes, _ := formatting.Decode(formatting.CB58, args.Signature)
	message, _ := formatting.Decode(formatting.CB58, args.Message)
	factory := crypto.FactorySECP256K1R{}
	pubkey, _ := factory.ToPublicKey(decodedPubkey)
	reply.Valid = pubkey.Verify(message, decodedSigBytes)
	return err

	/*
	var err error
	var pubkey crypto.PublicKey
	var expectedPubkey string
	var actualPubkey string
	var privkey crypto.PrivateKey
	var decodedPubkey []byte
	var decodedPrivkey []byte
	var decodedMsgBytes []byte
	var decodedSigBytes []byte
	factory := crypto.FactorySECP256K1R{}
	decodedPubkey, err = formatting.Decode(formatting.CB58, args.PublicKey)
	decodedPrivkey, err = formatting.Decode(formatting.CB58, args.PrivateKey)
	decodedSigBytes, err = formatting.Decode(formatting.CB58, args.Signature)
	privkey, err = factory.ToPrivateKey(decodedPrivkey)
	pubkey, err = factory.ToPublicKey(decodedPubkey)
	expectedPubkey, err= formatting.EncodeWithChecksum(formatting.CB58, privkey.PublicKey().Bytes())
	actualPubkey, err = formatting.EncodeWithChecksum(formatting.CB58, pubkey.Bytes())
	var sig []byte;
	var expected string
	var expectedChecksum string
	sig, err = privkey.Sign(decodedMsgBytes)
	reply.Valid = pubkey.Verify(messageBytes, decodedSigBytes)
	reply.Actual = args.Signature
	expected, err = formatting.EncodeWithoutChecksum(formatting.CB58, sig)
	expectedChecksum, err = formatting.EncodeWithChecksum(formatting.CB58, sig)
	reply.Expected = expected
	reply.ExpectedChecksum = expectedChecksum
	reply.ExpectedValid = pubkey.Verify(decodedMsgBytes, sig)
	reply.ExpectedPubkey = expectedPubkey
	reply.ActualPubkey = actualPubkey
	return err
	*/
}
