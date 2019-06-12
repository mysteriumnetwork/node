/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package generated

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// MystDEXABI is the input ABI used to generate the binding from.
const MystDEXABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"initialised\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newDestination\",\"type\":\"address\"}],\"name\":\"setFundsDestination\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"claimEthers\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"isOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"claimTokens\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getFundsDestination\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"token\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousDestination\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"newDestination\",\"type\":\"address\"}],\"name\":\"DestinationChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"constant\":false,\"inputs\":[{\"name\":\"_dexOwner\",\"type\":\"address\"},{\"name\":\"_token\",\"type\":\"address\"},{\"name\":\"_rate\",\"type\":\"uint256\"}],\"name\":\"initialise\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newRate\",\"type\":\"uint256\"}],\"name\":\"setRate\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"transferEthers\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"transferMyst\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// MystDEXBin is the compiled bytecode used for deploying new contracts.
const MystDEXBin = `0x6080604052336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a36112d2806100cf6000396000f3fe6080604052600436106100dd5760003560e01c80638595d1491161007f578063df8de3e711610059578063df8de3e714610600578063f2fde38b14610651578063f58c5b6e146106a2578063fc0c546a146106f9576100dd565b80638595d149146104ff5780638da5cb5b1461057a5780638f32d59b146105d1576100dd565b806334fcf437116100bb57806334fcf4371461043b57806338d2e411146104765780636931b550146104d1578063715018a6146104e8576100dd565b806307003bb4146103605780631254e64d1461038f578063238e130a146103ea575b600260149054906101000a900460ff1661015f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601b8152602001807f436f6e7472616374206973206e6f7420696e697469616c69736564000000000081525060200191505060405180910390fd5b6000610190670de0b6b3a76400006101826003543461075090919063ffffffff16565b61077690919063ffffffff16565b905080600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166370a08231306040518263ffffffff1660e01b8152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060206040518083038186803b15801561023257600080fd5b505afa158015610246573d6000803e3d6000fd5b505050506040513d602081101561025c57600080fd5b8101908080519060200190929190505050101561027857600080fd5b600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663a9059cbb33836040518363ffffffff1660e01b8152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050602060405180830381600087803b15801561032157600080fd5b505af1158015610335573d6000803e3d6000fd5b505050506040513d602081101561034b57600080fd5b81019080805190602001909291905050505050005b34801561036c57600080fd5b506103756107b0565b604051808215151515815260200191505060405180910390f35b34801561039b57600080fd5b506103e8600480360360408110156103b257600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506107c3565b005b3480156103f657600080fd5b506104396004803603602081101561040d57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610843565b005b34801561044757600080fd5b506104746004803603602081101561045e57600080fd5b810190808035906020019092919050505061094e565b005b34801561048257600080fd5b506104cf6004803603604081101561049957600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610969565b005b3480156104dd57600080fd5b506104e6610b49565b005b3480156104f457600080fd5b506104fd610c27565b005b34801561050b57600080fd5b506105786004803603606081101561052257600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610cf7565b005b34801561058657600080fd5b5061058f610deb565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156105dd57600080fd5b506105e6610e14565b604051808215151515815260200191505060405180910390f35b34801561060c57600080fd5b5061064f6004803603602081101561062357600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610e6b565b005b34801561065d57600080fd5b506106a06004803603602081101561067457600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611113565b005b3480156106ae57600080fd5b506106b7611130565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561070557600080fd5b5061070e61115a565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b600080821161075e57600080fd5b600082848161076957fe5b0490508091505092915050565b60008083141561078957600090506107aa565b600082840290508284828161079a57fe5b04146107a557600080fd5b809150505b92915050565b600260149054906101000a900460ff1681565b6107cb610e14565b6107d457600080fd5b803073ffffffffffffffffffffffffffffffffffffffff163110156107f857600080fd5b8173ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f1935050505015801561083e573d6000803e3d6000fd5b505050565b61084b610e14565b61085457600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141561088e57600080fd5b8073ffffffffffffffffffffffffffffffffffffffff16600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167fe1a66d77649cf0a57b9937073549f30f1c82bb865aaf066d2f299e37a62c6aad60405160405180910390a380600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b610956610e14565b61095f57600080fd5b8060038190555050565b610971610e14565b61097a57600080fd5b80600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166370a08231306040518263ffffffff1660e01b8152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060206040518083038186803b158015610a1a57600080fd5b505afa158015610a2e573d6000803e3d6000fd5b505050506040513d6020811015610a4457600080fd5b81019080805190602001909291905050501015610a6057600080fd5b600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663a9059cbb83836040518363ffffffff1660e01b8152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050602060405180830381600087803b158015610b0957600080fd5b505af1158015610b1d573d6000803e3d6000fd5b505050506040513d6020811015610b3357600080fd5b8101908080519060200190929190505050505050565b600073ffffffffffffffffffffffffffffffffffffffff16600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff161415610ba557600080fd5b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166108fc3073ffffffffffffffffffffffffffffffffffffffff16319081150290604051600060405180830381858888f19350505050158015610c24573d6000803e3d6000fd5b50565b610c2f610e14565b610c3857600080fd5b600073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a360008060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b600260149054906101000a900460ff1615610d7a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601f8152602001807f436f6e747261637420697320616c726561647920696e697469616c697365640081525060200191505060405180910390fd5b610d8383611180565b81600460006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550806003819055506001600260146101000a81548160ff021916908315150217905550505050565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614905090565b600073ffffffffffffffffffffffffffffffffffffffff16600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff161415610ec757600080fd5b600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff161415610f6e576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260258152602001806112796025913960400191505060405180910390fd5b60008173ffffffffffffffffffffffffffffffffffffffff166370a08231306040518263ffffffff1660e01b8152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060206040518083038186803b158015610fed57600080fd5b505afa158015611001573d6000803e3d6000fd5b505050506040513d602081101561101757600080fd5b810190808051906020019092919050505090508173ffffffffffffffffffffffffffffffffffffffff1663a9059cbb600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16836040518363ffffffff1660e01b8152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050602060405180830381600087803b1580156110d357600080fd5b505af11580156110e7573d6000803e3d6000fd5b505050506040513d60208110156110fd57600080fd5b8101908080519060200190929190505050505050565b61111b610e14565b61112457600080fd5b61112d81611180565b50565b6000600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614156111ba57600080fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505056fe6e617469766520746f6b656e2066756e64732063616e2774206265207265636f7665726564a265627a7a7230582087a6860153980c2bc73de6b7004e71389348cd1c0709731a0233c1df86fb71a664736f6c63430005090032`

// DeployMystDEX deploys a new Ethereum contract, binding an instance of MystDEX to it.
func DeployMystDEX(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MystDEX, error) {
	parsed, err := abi.JSON(strings.NewReader(MystDEXABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(MystDEXBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MystDEX{MystDEXCaller: MystDEXCaller{contract: contract}, MystDEXTransactor: MystDEXTransactor{contract: contract}, MystDEXFilterer: MystDEXFilterer{contract: contract}}, nil
}

// MystDEX is an auto generated Go binding around an Ethereum contract.
type MystDEX struct {
	MystDEXCaller     // Read-only binding to the contract
	MystDEXTransactor // Write-only binding to the contract
	MystDEXFilterer   // Log filterer for contract events
}

// MystDEXCaller is an auto generated read-only Go binding around an Ethereum contract.
type MystDEXCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MystDEXTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MystDEXTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MystDEXFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MystDEXFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MystDEXSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MystDEXSession struct {
	Contract     *MystDEX          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MystDEXCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MystDEXCallerSession struct {
	Contract *MystDEXCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// MystDEXTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MystDEXTransactorSession struct {
	Contract     *MystDEXTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// MystDEXRaw is an auto generated low-level Go binding around an Ethereum contract.
type MystDEXRaw struct {
	Contract *MystDEX // Generic contract binding to access the raw methods on
}

// MystDEXCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MystDEXCallerRaw struct {
	Contract *MystDEXCaller // Generic read-only contract binding to access the raw methods on
}

// MystDEXTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MystDEXTransactorRaw struct {
	Contract *MystDEXTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMystDEX creates a new instance of MystDEX, bound to a specific deployed contract.
func NewMystDEX(address common.Address, backend bind.ContractBackend) (*MystDEX, error) {
	contract, err := bindMystDEX(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MystDEX{MystDEXCaller: MystDEXCaller{contract: contract}, MystDEXTransactor: MystDEXTransactor{contract: contract}, MystDEXFilterer: MystDEXFilterer{contract: contract}}, nil
}

// NewMystDEXCaller creates a new read-only instance of MystDEX, bound to a specific deployed contract.
func NewMystDEXCaller(address common.Address, caller bind.ContractCaller) (*MystDEXCaller, error) {
	contract, err := bindMystDEX(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MystDEXCaller{contract: contract}, nil
}

// NewMystDEXTransactor creates a new write-only instance of MystDEX, bound to a specific deployed contract.
func NewMystDEXTransactor(address common.Address, transactor bind.ContractTransactor) (*MystDEXTransactor, error) {
	contract, err := bindMystDEX(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MystDEXTransactor{contract: contract}, nil
}

// NewMystDEXFilterer creates a new log filterer instance of MystDEX, bound to a specific deployed contract.
func NewMystDEXFilterer(address common.Address, filterer bind.ContractFilterer) (*MystDEXFilterer, error) {
	contract, err := bindMystDEX(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MystDEXFilterer{contract: contract}, nil
}

// bindMystDEX binds a generic wrapper to an already deployed contract.
func bindMystDEX(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MystDEXABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MystDEX *MystDEXRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _MystDEX.Contract.MystDEXCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MystDEX *MystDEXRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MystDEX.Contract.MystDEXTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MystDEX *MystDEXRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MystDEX.Contract.MystDEXTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MystDEX *MystDEXCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _MystDEX.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MystDEX *MystDEXTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MystDEX.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MystDEX *MystDEXTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MystDEX.Contract.contract.Transact(opts, method, params...)
}

// GetFundsDestination is a free data retrieval call binding the contract method 0xf58c5b6e.
//
// Solidity: function getFundsDestination() constant returns(address)
func (_MystDEX *MystDEXCaller) GetFundsDestination(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _MystDEX.contract.Call(opts, out, "getFundsDestination")
	return *ret0, err
}

// GetFundsDestination is a free data retrieval call binding the contract method 0xf58c5b6e.
//
// Solidity: function getFundsDestination() constant returns(address)
func (_MystDEX *MystDEXSession) GetFundsDestination() (common.Address, error) {
	return _MystDEX.Contract.GetFundsDestination(&_MystDEX.CallOpts)
}

// GetFundsDestination is a free data retrieval call binding the contract method 0xf58c5b6e.
//
// Solidity: function getFundsDestination() constant returns(address)
func (_MystDEX *MystDEXCallerSession) GetFundsDestination() (common.Address, error) {
	return _MystDEX.Contract.GetFundsDestination(&_MystDEX.CallOpts)
}

// Initialised is a free data retrieval call binding the contract method 0x07003bb4.
//
// Solidity: function initialised() constant returns(bool)
func (_MystDEX *MystDEXCaller) Initialised(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _MystDEX.contract.Call(opts, out, "initialised")
	return *ret0, err
}

// Initialised is a free data retrieval call binding the contract method 0x07003bb4.
//
// Solidity: function initialised() constant returns(bool)
func (_MystDEX *MystDEXSession) Initialised() (bool, error) {
	return _MystDEX.Contract.Initialised(&_MystDEX.CallOpts)
}

// Initialised is a free data retrieval call binding the contract method 0x07003bb4.
//
// Solidity: function initialised() constant returns(bool)
func (_MystDEX *MystDEXCallerSession) Initialised() (bool, error) {
	return _MystDEX.Contract.Initialised(&_MystDEX.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() constant returns(bool)
func (_MystDEX *MystDEXCaller) IsOwner(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _MystDEX.contract.Call(opts, out, "isOwner")
	return *ret0, err
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() constant returns(bool)
func (_MystDEX *MystDEXSession) IsOwner() (bool, error) {
	return _MystDEX.Contract.IsOwner(&_MystDEX.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() constant returns(bool)
func (_MystDEX *MystDEXCallerSession) IsOwner() (bool, error) {
	return _MystDEX.Contract.IsOwner(&_MystDEX.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_MystDEX *MystDEXCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _MystDEX.contract.Call(opts, out, "owner")
	return *ret0, err
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_MystDEX *MystDEXSession) Owner() (common.Address, error) {
	return _MystDEX.Contract.Owner(&_MystDEX.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_MystDEX *MystDEXCallerSession) Owner() (common.Address, error) {
	return _MystDEX.Contract.Owner(&_MystDEX.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() constant returns(address)
func (_MystDEX *MystDEXCaller) Token(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _MystDEX.contract.Call(opts, out, "token")
	return *ret0, err
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() constant returns(address)
func (_MystDEX *MystDEXSession) Token() (common.Address, error) {
	return _MystDEX.Contract.Token(&_MystDEX.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() constant returns(address)
func (_MystDEX *MystDEXCallerSession) Token() (common.Address, error) {
	return _MystDEX.Contract.Token(&_MystDEX.CallOpts)
}

// ClaimEthers is a paid mutator transaction binding the contract method 0x6931b550.
//
// Solidity: function claimEthers() returns()
func (_MystDEX *MystDEXTransactor) ClaimEthers(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MystDEX.contract.Transact(opts, "claimEthers")
}

// ClaimEthers is a paid mutator transaction binding the contract method 0x6931b550.
//
// Solidity: function claimEthers() returns()
func (_MystDEX *MystDEXSession) ClaimEthers() (*types.Transaction, error) {
	return _MystDEX.Contract.ClaimEthers(&_MystDEX.TransactOpts)
}

// ClaimEthers is a paid mutator transaction binding the contract method 0x6931b550.
//
// Solidity: function claimEthers() returns()
func (_MystDEX *MystDEXTransactorSession) ClaimEthers() (*types.Transaction, error) {
	return _MystDEX.Contract.ClaimEthers(&_MystDEX.TransactOpts)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_MystDEX *MystDEXTransactor) ClaimTokens(opts *bind.TransactOpts, _token common.Address) (*types.Transaction, error) {
	return _MystDEX.contract.Transact(opts, "claimTokens", _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_MystDEX *MystDEXSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _MystDEX.Contract.ClaimTokens(&_MystDEX.TransactOpts, _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_MystDEX *MystDEXTransactorSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _MystDEX.Contract.ClaimTokens(&_MystDEX.TransactOpts, _token)
}

// Initialise is a paid mutator transaction binding the contract method 0x8595d149.
//
// Solidity: function initialise(address _dexOwner, address _token, uint256 _rate) returns()
func (_MystDEX *MystDEXTransactor) Initialise(opts *bind.TransactOpts, _dexOwner common.Address, _token common.Address, _rate *big.Int) (*types.Transaction, error) {
	return _MystDEX.contract.Transact(opts, "initialise", _dexOwner, _token, _rate)
}

// Initialise is a paid mutator transaction binding the contract method 0x8595d149.
//
// Solidity: function initialise(address _dexOwner, address _token, uint256 _rate) returns()
func (_MystDEX *MystDEXSession) Initialise(_dexOwner common.Address, _token common.Address, _rate *big.Int) (*types.Transaction, error) {
	return _MystDEX.Contract.Initialise(&_MystDEX.TransactOpts, _dexOwner, _token, _rate)
}

// Initialise is a paid mutator transaction binding the contract method 0x8595d149.
//
// Solidity: function initialise(address _dexOwner, address _token, uint256 _rate) returns()
func (_MystDEX *MystDEXTransactorSession) Initialise(_dexOwner common.Address, _token common.Address, _rate *big.Int) (*types.Transaction, error) {
	return _MystDEX.Contract.Initialise(&_MystDEX.TransactOpts, _dexOwner, _token, _rate)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_MystDEX *MystDEXTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MystDEX.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_MystDEX *MystDEXSession) RenounceOwnership() (*types.Transaction, error) {
	return _MystDEX.Contract.RenounceOwnership(&_MystDEX.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_MystDEX *MystDEXTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _MystDEX.Contract.RenounceOwnership(&_MystDEX.TransactOpts)
}

// SetFundsDestination is a paid mutator transaction binding the contract method 0x238e130a.
//
// Solidity: function setFundsDestination(address _newDestination) returns()
func (_MystDEX *MystDEXTransactor) SetFundsDestination(opts *bind.TransactOpts, _newDestination common.Address) (*types.Transaction, error) {
	return _MystDEX.contract.Transact(opts, "setFundsDestination", _newDestination)
}

// SetFundsDestination is a paid mutator transaction binding the contract method 0x238e130a.
//
// Solidity: function setFundsDestination(address _newDestination) returns()
func (_MystDEX *MystDEXSession) SetFundsDestination(_newDestination common.Address) (*types.Transaction, error) {
	return _MystDEX.Contract.SetFundsDestination(&_MystDEX.TransactOpts, _newDestination)
}

// SetFundsDestination is a paid mutator transaction binding the contract method 0x238e130a.
//
// Solidity: function setFundsDestination(address _newDestination) returns()
func (_MystDEX *MystDEXTransactorSession) SetFundsDestination(_newDestination common.Address) (*types.Transaction, error) {
	return _MystDEX.Contract.SetFundsDestination(&_MystDEX.TransactOpts, _newDestination)
}

// SetRate is a paid mutator transaction binding the contract method 0x34fcf437.
//
// Solidity: function setRate(uint256 _newRate) returns()
func (_MystDEX *MystDEXTransactor) SetRate(opts *bind.TransactOpts, _newRate *big.Int) (*types.Transaction, error) {
	return _MystDEX.contract.Transact(opts, "setRate", _newRate)
}

// SetRate is a paid mutator transaction binding the contract method 0x34fcf437.
//
// Solidity: function setRate(uint256 _newRate) returns()
func (_MystDEX *MystDEXSession) SetRate(_newRate *big.Int) (*types.Transaction, error) {
	return _MystDEX.Contract.SetRate(&_MystDEX.TransactOpts, _newRate)
}

// SetRate is a paid mutator transaction binding the contract method 0x34fcf437.
//
// Solidity: function setRate(uint256 _newRate) returns()
func (_MystDEX *MystDEXTransactorSession) SetRate(_newRate *big.Int) (*types.Transaction, error) {
	return _MystDEX.Contract.SetRate(&_MystDEX.TransactOpts, _newRate)
}

// TransferEthers is a paid mutator transaction binding the contract method 0x1254e64d.
//
// Solidity: function transferEthers(address _to, uint256 _amount) returns()
func (_MystDEX *MystDEXTransactor) TransferEthers(opts *bind.TransactOpts, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MystDEX.contract.Transact(opts, "transferEthers", _to, _amount)
}

// TransferEthers is a paid mutator transaction binding the contract method 0x1254e64d.
//
// Solidity: function transferEthers(address _to, uint256 _amount) returns()
func (_MystDEX *MystDEXSession) TransferEthers(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MystDEX.Contract.TransferEthers(&_MystDEX.TransactOpts, _to, _amount)
}

// TransferEthers is a paid mutator transaction binding the contract method 0x1254e64d.
//
// Solidity: function transferEthers(address _to, uint256 _amount) returns()
func (_MystDEX *MystDEXTransactorSession) TransferEthers(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MystDEX.Contract.TransferEthers(&_MystDEX.TransactOpts, _to, _amount)
}

// TransferMyst is a paid mutator transaction binding the contract method 0x38d2e411.
//
// Solidity: function transferMyst(address _to, uint256 _amount) returns()
func (_MystDEX *MystDEXTransactor) TransferMyst(opts *bind.TransactOpts, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MystDEX.contract.Transact(opts, "transferMyst", _to, _amount)
}

// TransferMyst is a paid mutator transaction binding the contract method 0x38d2e411.
//
// Solidity: function transferMyst(address _to, uint256 _amount) returns()
func (_MystDEX *MystDEXSession) TransferMyst(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MystDEX.Contract.TransferMyst(&_MystDEX.TransactOpts, _to, _amount)
}

// TransferMyst is a paid mutator transaction binding the contract method 0x38d2e411.
//
// Solidity: function transferMyst(address _to, uint256 _amount) returns()
func (_MystDEX *MystDEXTransactorSession) TransferMyst(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MystDEX.Contract.TransferMyst(&_MystDEX.TransactOpts, _to, _amount)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_MystDEX *MystDEXTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _MystDEX.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_MystDEX *MystDEXSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _MystDEX.Contract.TransferOwnership(&_MystDEX.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_MystDEX *MystDEXTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _MystDEX.Contract.TransferOwnership(&_MystDEX.TransactOpts, newOwner)
}

// MystDEXDestinationChangedIterator is returned from FilterDestinationChanged and is used to iterate over the raw logs and unpacked data for DestinationChanged events raised by the MystDEX contract.
type MystDEXDestinationChangedIterator struct {
	Event *MystDEXDestinationChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MystDEXDestinationChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MystDEXDestinationChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MystDEXDestinationChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MystDEXDestinationChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MystDEXDestinationChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MystDEXDestinationChanged represents a DestinationChanged event raised by the MystDEX contract.
type MystDEXDestinationChanged struct {
	PreviousDestination common.Address
	NewDestination      common.Address
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterDestinationChanged is a free log retrieval operation binding the contract event 0xe1a66d77649cf0a57b9937073549f30f1c82bb865aaf066d2f299e37a62c6aad.
//
// Solidity: event DestinationChanged(address indexed previousDestination, address indexed newDestination)
func (_MystDEX *MystDEXFilterer) FilterDestinationChanged(opts *bind.FilterOpts, previousDestination []common.Address, newDestination []common.Address) (*MystDEXDestinationChangedIterator, error) {

	var previousDestinationRule []interface{}
	for _, previousDestinationItem := range previousDestination {
		previousDestinationRule = append(previousDestinationRule, previousDestinationItem)
	}
	var newDestinationRule []interface{}
	for _, newDestinationItem := range newDestination {
		newDestinationRule = append(newDestinationRule, newDestinationItem)
	}

	logs, sub, err := _MystDEX.contract.FilterLogs(opts, "DestinationChanged", previousDestinationRule, newDestinationRule)
	if err != nil {
		return nil, err
	}
	return &MystDEXDestinationChangedIterator{contract: _MystDEX.contract, event: "DestinationChanged", logs: logs, sub: sub}, nil
}

// WatchDestinationChanged is a free log subscription operation binding the contract event 0xe1a66d77649cf0a57b9937073549f30f1c82bb865aaf066d2f299e37a62c6aad.
//
// Solidity: event DestinationChanged(address indexed previousDestination, address indexed newDestination)
func (_MystDEX *MystDEXFilterer) WatchDestinationChanged(opts *bind.WatchOpts, sink chan<- *MystDEXDestinationChanged, previousDestination []common.Address, newDestination []common.Address) (event.Subscription, error) {

	var previousDestinationRule []interface{}
	for _, previousDestinationItem := range previousDestination {
		previousDestinationRule = append(previousDestinationRule, previousDestinationItem)
	}
	var newDestinationRule []interface{}
	for _, newDestinationItem := range newDestination {
		newDestinationRule = append(newDestinationRule, newDestinationItem)
	}

	logs, sub, err := _MystDEX.contract.WatchLogs(opts, "DestinationChanged", previousDestinationRule, newDestinationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MystDEXDestinationChanged)
				if err := _MystDEX.contract.UnpackLog(event, "DestinationChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// MystDEXOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the MystDEX contract.
type MystDEXOwnershipTransferredIterator struct {
	Event *MystDEXOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MystDEXOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MystDEXOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MystDEXOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MystDEXOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MystDEXOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MystDEXOwnershipTransferred represents a OwnershipTransferred event raised by the MystDEX contract.
type MystDEXOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_MystDEX *MystDEXFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*MystDEXOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _MystDEX.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &MystDEXOwnershipTransferredIterator{contract: _MystDEX.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_MystDEX *MystDEXFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *MystDEXOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _MystDEX.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MystDEXOwnershipTransferred)
				if err := _MystDEX.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}
