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

// IdentityImplementationABI is the input ABI used to generate the binding from.
const IdentityImplementationABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"identityHash\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newDestination\",\"type\":\"address\"}],\"name\":\"setFundsDestination\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"paidAmounts\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"dex\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"claimEthers\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"isOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"}],\"name\":\"claimTokens\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getFundsDestination\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"token\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"},{\"name\":\"_DEXImplementation\",\"type\":\"address\"},{\"name\":\"_DEXOwner\",\"type\":\"address\"},{\"name\":\"_rate\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"receiver\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"totalBalance\",\"type\":\"uint256\"}],\"name\":\"Withdrawn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"receiver\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"totalSettled\",\"type\":\"uint256\"}],\"name\":\"PromiseSettled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousDestination\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"newDestination\",\"type\":\"address\"}],\"name\":\"DestinationChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"constant\":false,\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"},{\"name\":\"_identityHash\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"isInitialized\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"},{\"name\":\"_fee\",\"type\":\"uint256\"},{\"name\":\"_extraDataHash\",\"type\":\"bytes32\"},{\"name\":\"_signature\",\"type\":\"bytes\"}],\"name\":\"settlePromise\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"},{\"name\":\"_signature\",\"type\":\"bytes\"}],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newDestination\",\"type\":\"address\"},{\"name\":\"_signature\",\"type\":\"bytes\"}],\"name\":\"setFundsDestinationByCheque\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// IdentityImplementationBin is the compiled bytecode used for deploying new contracts.
const IdentityImplementationBin = `0x60806040523480156200001157600080fd5b5060405160808062002b18833981018060405260808110156200003357600080fd5b8101908080519060200190929190805190602001909291908051906020019092919080519060200190929190505050336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3600073ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff1614156200015957600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1614156200019457600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff161415620001cf57600080fd5b8282604051620001df90620003a7565b808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200192505050604051809103906000f08015801562000265573d6000803e3d6000fd5b50600460006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16638595d1498386846040518463ffffffff1660e01b8152600401808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019350505050600060405180830381600087803b1580156200038457600080fd5b505af115801562000399573d6000803e3d6000fd5b5050505050505050620003b5565b61086b80620022ad83390190565b611ee880620003c56000396000f3fe6080604052600436106100fe5760003560e01c80636931b550116100955780638f32d59b116100645780638f32d59b1461077b578063df8de3e7146107aa578063f2fde38b146107fb578063f58c5b6e1461084c578063fc0c546a146108a3576100fe565b80636931b5501461060e5780636a2b76ad14610625578063715018a61461070d5780638da5cb5b14610724576100fe565b8063392e53cd116100d1578063392e53cd14610411578063485cc9551461044057806348d9f01e146104b1578063692058c2146105b7576100fe565b8063212ff20a14610212578063238e130a1461026957806324e7c4b7146102ba57806331f092651461031f575b60006060600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163460003660405180838380828437808301925050509250505060006040518083038185875af1925050503d8060008114610191576040519150601f19603f3d011682016040523d82523d6000602084013e610196565b606091505b50915091508161020e576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260168152602001807f5478207761732072656a6563746564206279204445580000000000000000000081525060200191505060405180910390fd5b5050005b34801561021e57600080fd5b506102276108fa565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561027557600080fd5b506102b86004803603602081101561028c57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610920565b005b3480156102c657600080fd5b50610309600480360360208110156102dd57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610a2b565b6040518082815260200191505060405180910390f35b34801561032b57600080fd5b5061040f6004803603606081101561034257600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291908035906020019064010000000081111561038957600080fd5b82018360208201111561039b57600080fd5b803590602001918460018302840111640100000000831117156103bd57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050509192919290505050610a43565b005b34801561041d57600080fd5b50610426610ba0565b604051808215151515815260200191505060405180910390f35b34801561044c57600080fd5b506104af6004803603604081101561046357600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610bf9565b005b3480156104bd57600080fd5b506105b5600480360360a08110156104d457600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019092919080359060200190929190803590602001909291908035906020019064010000000081111561052f57600080fd5b82018360208201111561054157600080fd5b8035906020019184600183028401116401000000008311171561056357600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050509192919290505050610d6e565b005b3480156105c357600080fd5b506105cc610e1c565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561061a57600080fd5b50610623610e42565b005b34801561063157600080fd5b5061070b6004803603604081101561064857600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019064010000000081111561068557600080fd5b82018360208201111561069757600080fd5b803590602001918460018302840111640100000000831117156106b957600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050509192919290505050610f20565b005b34801561071957600080fd5b506107226111b6565b005b34801561073057600080fd5b50610739611286565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561078757600080fd5b506107906112af565b604051808215151515815260200191505060405180910390f35b3480156107b657600080fd5b506107f9600480360360208110156107cd57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611306565b005b34801561080757600080fd5b5061084a6004803603602081101561081e57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611507565b005b34801561085857600080fd5b50610861611524565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156108af57600080fd5b506108b861154e565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6109286112af565b61093157600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141561096b57600080fd5b8073ffffffffffffffffffffffffffffffffffffffff16600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167fe1a66d77649cf0a57b9937073549f30f1c82bb865aaf066d2f299e37a62c6aad60405160405180910390a380600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b60056020528060005260406000206000915090505481565b6000604051806000019050604051809103902090506000610a68858560008587611574565b90508473ffffffffffffffffffffffffffffffffffffffff167f92ccf450a286a957af52509bc1c9939d1a6a481783e142e41e2499f0bb66ebc682600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166370a08231306040518263ffffffff1660e01b8152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060206040518083038186803b158015610b4257600080fd5b505afa158015610b56573d6000803e3d6000fd5b505050506040513d6020811015610b6c57600080fd5b8101908080519060200190929190505050604051808381526020018281526020019250505060405180910390a25050505050565b60008073ffffffffffffffffffffffffffffffffffffffff16600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff161415905090565b610c01610ba0565b15610c74576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260168152602001807f497320616c726561647920696e697469616c697a65640000000000000000000081525060200191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff161415610cae57600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff161415610ce857600080fd5b81600260006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555080600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505050565b6000610d7d8686868686611574565b90508573ffffffffffffffffffffffffffffffffffffffff167f50c3491624aa1825a7653df63d067fecd5c8634ba63c99c4a7cf04ff1436070b82600560008a73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054604051808381526020018281526020019250505060405180910390a2505050505050565b600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600073ffffffffffffffffffffffffffffffffffffffff16600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff161415610e9e57600080fd5b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166108fc3073ffffffffffffffffffffffffffffffffffffffff16319081150290604051600060405180830381858888f19350505050158015610f1d573d6000803e3d6000fd5b50565b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff161415610f5a57600080fd5b600061104c826040518060400160405280601681526020017f5365742066756e64732064657374696e6174696f6e3a00000000000000000000815250856040516020018083805190602001908083835b60208310610fcd5780518252602082019150602081019050602083039250610faa565b6001836020036101000a0380198251168184511680821785525050505050509050018273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1660601b81526014019250505060405160208183030381529060405280519060200120611c5d90919063ffffffff16565b9050600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16146110f4576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526024815260200180611e996024913960400191505060405180910390fd5b8273ffffffffffffffffffffffffffffffffffffffff16600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167fe1a66d77649cf0a57b9937073549f30f1c82bb865aaf066d2f299e37a62c6aad60405160405180910390a382600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505050565b6111be6112af565b6111c757600080fd5b600073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a360008060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614905090565b600073ffffffffffffffffffffffffffffffffffffffff16600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141561136257600080fd5b60008173ffffffffffffffffffffffffffffffffffffffff166370a08231306040518263ffffffff1660e01b8152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060206040518083038186803b1580156113e157600080fd5b505afa1580156113f5573d6000803e3d6000fd5b505050506040513d602081101561140b57600080fd5b810190808051906020019092919050505090508173ffffffffffffffffffffffffffffffffffffffff1663a9059cbb600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16836040518363ffffffff1660e01b8152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050602060405180830381600087803b1580156114c757600080fd5b505af11580156114db573d6000803e3d6000fd5b505050506040513d60208110156114f157600080fd5b8101908080519060200190929190505050505050565b61150f6112af565b61151857600080fd5b61152181611d3f565b50565b6000600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60008073ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff1614156115af57600080fd5b60006116b9836040518060400160405280601381526020017f536574746c656d656e7420726571756573743a00000000000000000000000000815250898989896040516020018086805190602001908083835b602083106116255780518252602082019150602081019050602083039250611602565b6001836020036101000a0380198251168184511680821785525050505050509050018573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1660601b81526014018481526020018381526020018281526020019550505050505060405160208183030381529060405280519060200120611c5d90919063ffffffff16565b9050600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614611761576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526024815260200180611e996024913960400191505060405180910390fd5b60006117b5600560008a73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205488611e3790919063ffffffff16565b905060008111611810576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526022815260200180611e776022913960400191505060405180910390fd5b600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166370a08231306040518263ffffffff1660e01b8152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060206040518083038186803b1580156118af57600080fd5b505afa1580156118c3573d6000803e3d6000fd5b505050506040513d60208110156118d957600080fd5b81019080805190602001909291905050508111156119ce57600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166370a08231306040518263ffffffff1660e01b8152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060206040518083038186803b15801561199057600080fd5b505afa1580156119a4573d6000803e3d6000fd5b505050506040513d60208110156119ba57600080fd5b810190808051906020019092919050505090505b6000611a2282600560008c73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054611e5790919063ffffffff16565b905080600560008b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663a9059cbb8a611aba8a86611e3790919063ffffffff16565b6040518363ffffffff1660e01b8152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050602060405180830381600087803b158015611b2357600080fd5b505af1158015611b37573d6000803e3d6000fd5b505050506040513d6020811015611b4d57600080fd5b8101908080519060200190929190505050506000871115611c4e57600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663a9059cbb33896040518363ffffffff1660e01b8152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050602060405180830381600087803b158015611c1157600080fd5b505af1158015611c25573d6000803e3d6000fd5b505050506040513d6020811015611c3b57600080fd5b8101908080519060200190929190505050505b81935050505095945050505050565b6000806000806041855114611c785760009350505050611d39565b6020850151925060408501519150606085015160001a9050601b8160ff161015611ca357601b810190505b601b8160ff1614158015611cbb5750601c8160ff1614155b15611ccc5760009350505050611d39565b60018682858560405160008152602001604052604051808581526020018460ff1660ff1681526020018381526020018281526020019450505050506020604051602081039080840390855afa158015611d29573d6000803e3d6000fd5b5050506020604051035193505050505b92915050565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff161415611d7957600080fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b600082821115611e4657600080fd5b600082840390508091505092915050565b600080828401905083811015611e6c57600080fd5b809150509291505056fe416d6f756e7420746f20736574746c65206973206c6573732074686174207a65726f4861766520746f206265207369676e65642062792070726f706572206964656e74697479a165627a7a72305820e1b3ff85220a65055c0d1434b94caef68fa4c3c6f016448f044e59ec24aaf7f80029608060405234801561001057600080fd5b5060405160408061086b8339810180604052604081101561003057600080fd5b810190808051906020019092919080519060200190929190505050600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16141561008557600080fd5b600060405180807f4d79737444455850726f78792e6f776e65720000000000000000000000000000815250601201905060405180910390209050600060405180807f4d797344455850726f78792e696d706c656d656e746174696f6e000000000000815250601a0190506040518091039020905082825583815550505050610759806101126000396000f3fe60806040526004361061004a5760003560e01c806309b17085146100815780636f127677146100d85780638fd13adc1461012f578063c7f846051461020a578063fbd707681461025b575b60006100546102ac565b905060405136600082376000803683856127105a03f43d806000843e816000811461007d578184f35b8184fd5b34801561008d57600080fd5b506100966102ef565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156100e457600080fd5b506100ed6102ac565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6102086004803603604081101561014557600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019064010000000081111561018257600080fd5b82018360208201111561019457600080fd5b803590602001918460018302840111640100000000831117156101b657600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050509192919290505050610332565b005b34801561021657600080fd5b506102596004803603602081101561022d57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610518565b005b34801561026757600080fd5b506102aa6004803603602081101561027e57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610644565b005b60008060405180807f4d797344455850726f78792e696d706c656d656e746174696f6e000000000000815250601a01905060405180910390209050805491505090565b60008060405180807f4d79737444455850726f78792e6f776e65720000000000000000000000000000815250601201905060405180910390209050805491505090565b61033a6102ef565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146103da576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260208152602001807f4f6e6c79206f776e65722063616e2072756e20746869732066756e6374696f6e81525060200191505060405180910390fd5b6103e382610518565b600060603073ffffffffffffffffffffffffffffffffffffffff1634846040518082805190602001908083835b602083106104335780518252602082019150602081019050602083039250610410565b6001836020036101000a03801982511681845116808217855250505050505090500191505060006040518083038185875af1925050503d8060008114610495576040519150601f19603f3d011682016040523d82523d6000602084013e61049a565b606091505b509150915081610512576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260198152602001807f43616c6c696e67206e657720746172676574206661696c65640000000000000081525060200191505060405180910390fd5b50505050565b6105206102ef565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146105c0576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260208152602001807f4f6e6c79206f776e65722063616e2072756e20746869732066756e6374696f6e81525060200191505060405180910390fd5b600060405180807f4d797344455850726f78792e696d706c656d656e746174696f6e000000000000815250601a019050604051809103902090508181558173ffffffffffffffffffffffffffffffffffffffff167fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b60405160405180910390a25050565b61064c6102ef565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146106ec576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260208152602001807f4f6e6c79206f776e65722063616e2072756e20746869732066756e6374696f6e81525060200191505060405180910390fd5b600060405180807f4d79737444455850726f78792e6f776e65720000000000000000000000000000815250601201905060405180910390209050818155505056fea165627a7a723058204082ff42e2fc66f0b24c4d9d1df6d9ebc72b3d59e7f37a1b9e361408bceaa5d60029`

// DeployIdentityImplementation deploys a new Ethereum contract, binding an instance of IdentityImplementation to it.
func DeployIdentityImplementation(auth *bind.TransactOpts, backend bind.ContractBackend, _token common.Address, _DEXImplementation common.Address, _DEXOwner common.Address, _rate *big.Int) (common.Address, *types.Transaction, *IdentityImplementation, error) {
	parsed, err := abi.JSON(strings.NewReader(IdentityImplementationABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(IdentityImplementationBin), backend, _token, _DEXImplementation, _DEXOwner, _rate)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &IdentityImplementation{IdentityImplementationCaller: IdentityImplementationCaller{contract: contract}, IdentityImplementationTransactor: IdentityImplementationTransactor{contract: contract}, IdentityImplementationFilterer: IdentityImplementationFilterer{contract: contract}}, nil
}

// IdentityImplementation is an auto generated Go binding around an Ethereum contract.
type IdentityImplementation struct {
	IdentityImplementationCaller     // Read-only binding to the contract
	IdentityImplementationTransactor // Write-only binding to the contract
	IdentityImplementationFilterer   // Log filterer for contract events
}

// IdentityImplementationCaller is an auto generated read-only Go binding around an Ethereum contract.
type IdentityImplementationCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IdentityImplementationTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IdentityImplementationTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IdentityImplementationFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IdentityImplementationFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IdentityImplementationSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IdentityImplementationSession struct {
	Contract     *IdentityImplementation // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// IdentityImplementationCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IdentityImplementationCallerSession struct {
	Contract *IdentityImplementationCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// IdentityImplementationTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IdentityImplementationTransactorSession struct {
	Contract     *IdentityImplementationTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// IdentityImplementationRaw is an auto generated low-level Go binding around an Ethereum contract.
type IdentityImplementationRaw struct {
	Contract *IdentityImplementation // Generic contract binding to access the raw methods on
}

// IdentityImplementationCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IdentityImplementationCallerRaw struct {
	Contract *IdentityImplementationCaller // Generic read-only contract binding to access the raw methods on
}

// IdentityImplementationTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IdentityImplementationTransactorRaw struct {
	Contract *IdentityImplementationTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIdentityImplementation creates a new instance of IdentityImplementation, bound to a specific deployed contract.
func NewIdentityImplementation(address common.Address, backend bind.ContractBackend) (*IdentityImplementation, error) {
	contract, err := bindIdentityImplementation(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IdentityImplementation{IdentityImplementationCaller: IdentityImplementationCaller{contract: contract}, IdentityImplementationTransactor: IdentityImplementationTransactor{contract: contract}, IdentityImplementationFilterer: IdentityImplementationFilterer{contract: contract}}, nil
}

// NewIdentityImplementationCaller creates a new read-only instance of IdentityImplementation, bound to a specific deployed contract.
func NewIdentityImplementationCaller(address common.Address, caller bind.ContractCaller) (*IdentityImplementationCaller, error) {
	contract, err := bindIdentityImplementation(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IdentityImplementationCaller{contract: contract}, nil
}

// NewIdentityImplementationTransactor creates a new write-only instance of IdentityImplementation, bound to a specific deployed contract.
func NewIdentityImplementationTransactor(address common.Address, transactor bind.ContractTransactor) (*IdentityImplementationTransactor, error) {
	contract, err := bindIdentityImplementation(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IdentityImplementationTransactor{contract: contract}, nil
}

// NewIdentityImplementationFilterer creates a new log filterer instance of IdentityImplementation, bound to a specific deployed contract.
func NewIdentityImplementationFilterer(address common.Address, filterer bind.ContractFilterer) (*IdentityImplementationFilterer, error) {
	contract, err := bindIdentityImplementation(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IdentityImplementationFilterer{contract: contract}, nil
}

// bindIdentityImplementation binds a generic wrapper to an already deployed contract.
func bindIdentityImplementation(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(IdentityImplementationABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IdentityImplementation *IdentityImplementationRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _IdentityImplementation.Contract.IdentityImplementationCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IdentityImplementation *IdentityImplementationRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.IdentityImplementationTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IdentityImplementation *IdentityImplementationRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.IdentityImplementationTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IdentityImplementation *IdentityImplementationCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _IdentityImplementation.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IdentityImplementation *IdentityImplementationTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IdentityImplementation *IdentityImplementationTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.contract.Transact(opts, method, params...)
}

// Dex is a free data retrieval call binding the contract method 0x692058c2.
//
// Solidity: function dex() constant returns(address)
func (_IdentityImplementation *IdentityImplementationCaller) Dex(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _IdentityImplementation.contract.Call(opts, out, "dex")
	return *ret0, err
}

// Dex is a free data retrieval call binding the contract method 0x692058c2.
//
// Solidity: function dex() constant returns(address)
func (_IdentityImplementation *IdentityImplementationSession) Dex() (common.Address, error) {
	return _IdentityImplementation.Contract.Dex(&_IdentityImplementation.CallOpts)
}

// Dex is a free data retrieval call binding the contract method 0x692058c2.
//
// Solidity: function dex() constant returns(address)
func (_IdentityImplementation *IdentityImplementationCallerSession) Dex() (common.Address, error) {
	return _IdentityImplementation.Contract.Dex(&_IdentityImplementation.CallOpts)
}

// GetFundsDestination is a free data retrieval call binding the contract method 0xf58c5b6e.
//
// Solidity: function getFundsDestination() constant returns(address)
func (_IdentityImplementation *IdentityImplementationCaller) GetFundsDestination(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _IdentityImplementation.contract.Call(opts, out, "getFundsDestination")
	return *ret0, err
}

// GetFundsDestination is a free data retrieval call binding the contract method 0xf58c5b6e.
//
// Solidity: function getFundsDestination() constant returns(address)
func (_IdentityImplementation *IdentityImplementationSession) GetFundsDestination() (common.Address, error) {
	return _IdentityImplementation.Contract.GetFundsDestination(&_IdentityImplementation.CallOpts)
}

// GetFundsDestination is a free data retrieval call binding the contract method 0xf58c5b6e.
//
// Solidity: function getFundsDestination() constant returns(address)
func (_IdentityImplementation *IdentityImplementationCallerSession) GetFundsDestination() (common.Address, error) {
	return _IdentityImplementation.Contract.GetFundsDestination(&_IdentityImplementation.CallOpts)
}

// IdentityHash is a free data retrieval call binding the contract method 0x212ff20a.
//
// Solidity: function identityHash() constant returns(address)
func (_IdentityImplementation *IdentityImplementationCaller) IdentityHash(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _IdentityImplementation.contract.Call(opts, out, "identityHash")
	return *ret0, err
}

// IdentityHash is a free data retrieval call binding the contract method 0x212ff20a.
//
// Solidity: function identityHash() constant returns(address)
func (_IdentityImplementation *IdentityImplementationSession) IdentityHash() (common.Address, error) {
	return _IdentityImplementation.Contract.IdentityHash(&_IdentityImplementation.CallOpts)
}

// IdentityHash is a free data retrieval call binding the contract method 0x212ff20a.
//
// Solidity: function identityHash() constant returns(address)
func (_IdentityImplementation *IdentityImplementationCallerSession) IdentityHash() (common.Address, error) {
	return _IdentityImplementation.Contract.IdentityHash(&_IdentityImplementation.CallOpts)
}

// IsInitialized is a free data retrieval call binding the contract method 0x392e53cd.
//
// Solidity: function isInitialized() constant returns(bool)
func (_IdentityImplementation *IdentityImplementationCaller) IsInitialized(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _IdentityImplementation.contract.Call(opts, out, "isInitialized")
	return *ret0, err
}

// IsInitialized is a free data retrieval call binding the contract method 0x392e53cd.
//
// Solidity: function isInitialized() constant returns(bool)
func (_IdentityImplementation *IdentityImplementationSession) IsInitialized() (bool, error) {
	return _IdentityImplementation.Contract.IsInitialized(&_IdentityImplementation.CallOpts)
}

// IsInitialized is a free data retrieval call binding the contract method 0x392e53cd.
//
// Solidity: function isInitialized() constant returns(bool)
func (_IdentityImplementation *IdentityImplementationCallerSession) IsInitialized() (bool, error) {
	return _IdentityImplementation.Contract.IsInitialized(&_IdentityImplementation.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() constant returns(bool)
func (_IdentityImplementation *IdentityImplementationCaller) IsOwner(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _IdentityImplementation.contract.Call(opts, out, "isOwner")
	return *ret0, err
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() constant returns(bool)
func (_IdentityImplementation *IdentityImplementationSession) IsOwner() (bool, error) {
	return _IdentityImplementation.Contract.IsOwner(&_IdentityImplementation.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() constant returns(bool)
func (_IdentityImplementation *IdentityImplementationCallerSession) IsOwner() (bool, error) {
	return _IdentityImplementation.Contract.IsOwner(&_IdentityImplementation.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_IdentityImplementation *IdentityImplementationCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _IdentityImplementation.contract.Call(opts, out, "owner")
	return *ret0, err
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_IdentityImplementation *IdentityImplementationSession) Owner() (common.Address, error) {
	return _IdentityImplementation.Contract.Owner(&_IdentityImplementation.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_IdentityImplementation *IdentityImplementationCallerSession) Owner() (common.Address, error) {
	return _IdentityImplementation.Contract.Owner(&_IdentityImplementation.CallOpts)
}

// PaidAmounts is a free data retrieval call binding the contract method 0x24e7c4b7.
//
// Solidity: function paidAmounts(address ) constant returns(uint256)
func (_IdentityImplementation *IdentityImplementationCaller) PaidAmounts(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _IdentityImplementation.contract.Call(opts, out, "paidAmounts", arg0)
	return *ret0, err
}

// PaidAmounts is a free data retrieval call binding the contract method 0x24e7c4b7.
//
// Solidity: function paidAmounts(address ) constant returns(uint256)
func (_IdentityImplementation *IdentityImplementationSession) PaidAmounts(arg0 common.Address) (*big.Int, error) {
	return _IdentityImplementation.Contract.PaidAmounts(&_IdentityImplementation.CallOpts, arg0)
}

// PaidAmounts is a free data retrieval call binding the contract method 0x24e7c4b7.
//
// Solidity: function paidAmounts(address ) constant returns(uint256)
func (_IdentityImplementation *IdentityImplementationCallerSession) PaidAmounts(arg0 common.Address) (*big.Int, error) {
	return _IdentityImplementation.Contract.PaidAmounts(&_IdentityImplementation.CallOpts, arg0)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() constant returns(address)
func (_IdentityImplementation *IdentityImplementationCaller) Token(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _IdentityImplementation.contract.Call(opts, out, "token")
	return *ret0, err
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() constant returns(address)
func (_IdentityImplementation *IdentityImplementationSession) Token() (common.Address, error) {
	return _IdentityImplementation.Contract.Token(&_IdentityImplementation.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() constant returns(address)
func (_IdentityImplementation *IdentityImplementationCallerSession) Token() (common.Address, error) {
	return _IdentityImplementation.Contract.Token(&_IdentityImplementation.CallOpts)
}

// ClaimEthers is a paid mutator transaction binding the contract method 0x6931b550.
//
// Solidity: function claimEthers() returns()
func (_IdentityImplementation *IdentityImplementationTransactor) ClaimEthers(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IdentityImplementation.contract.Transact(opts, "claimEthers")
}

// ClaimEthers is a paid mutator transaction binding the contract method 0x6931b550.
//
// Solidity: function claimEthers() returns()
func (_IdentityImplementation *IdentityImplementationSession) ClaimEthers() (*types.Transaction, error) {
	return _IdentityImplementation.Contract.ClaimEthers(&_IdentityImplementation.TransactOpts)
}

// ClaimEthers is a paid mutator transaction binding the contract method 0x6931b550.
//
// Solidity: function claimEthers() returns()
func (_IdentityImplementation *IdentityImplementationTransactorSession) ClaimEthers() (*types.Transaction, error) {
	return _IdentityImplementation.Contract.ClaimEthers(&_IdentityImplementation.TransactOpts)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address token) returns()
func (_IdentityImplementation *IdentityImplementationTransactor) ClaimTokens(opts *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.contract.Transact(opts, "claimTokens", token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address token) returns()
func (_IdentityImplementation *IdentityImplementationSession) ClaimTokens(token common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.ClaimTokens(&_IdentityImplementation.TransactOpts, token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address token) returns()
func (_IdentityImplementation *IdentityImplementationTransactorSession) ClaimTokens(token common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.ClaimTokens(&_IdentityImplementation.TransactOpts, token)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _token, address _identityHash) returns()
func (_IdentityImplementation *IdentityImplementationTransactor) Initialize(opts *bind.TransactOpts, _token common.Address, _identityHash common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.contract.Transact(opts, "initialize", _token, _identityHash)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _token, address _identityHash) returns()
func (_IdentityImplementation *IdentityImplementationSession) Initialize(_token common.Address, _identityHash common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.Initialize(&_IdentityImplementation.TransactOpts, _token, _identityHash)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _token, address _identityHash) returns()
func (_IdentityImplementation *IdentityImplementationTransactorSession) Initialize(_token common.Address, _identityHash common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.Initialize(&_IdentityImplementation.TransactOpts, _token, _identityHash)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_IdentityImplementation *IdentityImplementationTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IdentityImplementation.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_IdentityImplementation *IdentityImplementationSession) RenounceOwnership() (*types.Transaction, error) {
	return _IdentityImplementation.Contract.RenounceOwnership(&_IdentityImplementation.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_IdentityImplementation *IdentityImplementationTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _IdentityImplementation.Contract.RenounceOwnership(&_IdentityImplementation.TransactOpts)
}

// SetFundsDestination is a paid mutator transaction binding the contract method 0x238e130a.
//
// Solidity: function setFundsDestination(address _newDestination) returns()
func (_IdentityImplementation *IdentityImplementationTransactor) SetFundsDestination(opts *bind.TransactOpts, _newDestination common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.contract.Transact(opts, "setFundsDestination", _newDestination)
}

// SetFundsDestination is a paid mutator transaction binding the contract method 0x238e130a.
//
// Solidity: function setFundsDestination(address _newDestination) returns()
func (_IdentityImplementation *IdentityImplementationSession) SetFundsDestination(_newDestination common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.SetFundsDestination(&_IdentityImplementation.TransactOpts, _newDestination)
}

// SetFundsDestination is a paid mutator transaction binding the contract method 0x238e130a.
//
// Solidity: function setFundsDestination(address _newDestination) returns()
func (_IdentityImplementation *IdentityImplementationTransactorSession) SetFundsDestination(_newDestination common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.SetFundsDestination(&_IdentityImplementation.TransactOpts, _newDestination)
}

// SetFundsDestinationByCheque is a paid mutator transaction binding the contract method 0x6a2b76ad.
//
// Solidity: function setFundsDestinationByCheque(address _newDestination, bytes _signature) returns()
func (_IdentityImplementation *IdentityImplementationTransactor) SetFundsDestinationByCheque(opts *bind.TransactOpts, _newDestination common.Address, _signature []byte) (*types.Transaction, error) {
	return _IdentityImplementation.contract.Transact(opts, "setFundsDestinationByCheque", _newDestination, _signature)
}

// SetFundsDestinationByCheque is a paid mutator transaction binding the contract method 0x6a2b76ad.
//
// Solidity: function setFundsDestinationByCheque(address _newDestination, bytes _signature) returns()
func (_IdentityImplementation *IdentityImplementationSession) SetFundsDestinationByCheque(_newDestination common.Address, _signature []byte) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.SetFundsDestinationByCheque(&_IdentityImplementation.TransactOpts, _newDestination, _signature)
}

// SetFundsDestinationByCheque is a paid mutator transaction binding the contract method 0x6a2b76ad.
//
// Solidity: function setFundsDestinationByCheque(address _newDestination, bytes _signature) returns()
func (_IdentityImplementation *IdentityImplementationTransactorSession) SetFundsDestinationByCheque(_newDestination common.Address, _signature []byte) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.SetFundsDestinationByCheque(&_IdentityImplementation.TransactOpts, _newDestination, _signature)
}

// SettlePromise is a paid mutator transaction binding the contract method 0x48d9f01e.
//
// Solidity: function settlePromise(address _to, uint256 _amount, uint256 _fee, bytes32 _extraDataHash, bytes _signature) returns()
func (_IdentityImplementation *IdentityImplementationTransactor) SettlePromise(opts *bind.TransactOpts, _to common.Address, _amount *big.Int, _fee *big.Int, _extraDataHash [32]byte, _signature []byte) (*types.Transaction, error) {
	return _IdentityImplementation.contract.Transact(opts, "settlePromise", _to, _amount, _fee, _extraDataHash, _signature)
}

// SettlePromise is a paid mutator transaction binding the contract method 0x48d9f01e.
//
// Solidity: function settlePromise(address _to, uint256 _amount, uint256 _fee, bytes32 _extraDataHash, bytes _signature) returns()
func (_IdentityImplementation *IdentityImplementationSession) SettlePromise(_to common.Address, _amount *big.Int, _fee *big.Int, _extraDataHash [32]byte, _signature []byte) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.SettlePromise(&_IdentityImplementation.TransactOpts, _to, _amount, _fee, _extraDataHash, _signature)
}

// SettlePromise is a paid mutator transaction binding the contract method 0x48d9f01e.
//
// Solidity: function settlePromise(address _to, uint256 _amount, uint256 _fee, bytes32 _extraDataHash, bytes _signature) returns()
func (_IdentityImplementation *IdentityImplementationTransactorSession) SettlePromise(_to common.Address, _amount *big.Int, _fee *big.Int, _extraDataHash [32]byte, _signature []byte) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.SettlePromise(&_IdentityImplementation.TransactOpts, _to, _amount, _fee, _extraDataHash, _signature)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_IdentityImplementation *IdentityImplementationTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_IdentityImplementation *IdentityImplementationSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.TransferOwnership(&_IdentityImplementation.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_IdentityImplementation *IdentityImplementationTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.TransferOwnership(&_IdentityImplementation.TransactOpts, newOwner)
}

// Withdraw is a paid mutator transaction binding the contract method 0x31f09265.
//
// Solidity: function withdraw(address _to, uint256 _amount, bytes _signature) returns()
func (_IdentityImplementation *IdentityImplementationTransactor) Withdraw(opts *bind.TransactOpts, _to common.Address, _amount *big.Int, _signature []byte) (*types.Transaction, error) {
	return _IdentityImplementation.contract.Transact(opts, "withdraw", _to, _amount, _signature)
}

// Withdraw is a paid mutator transaction binding the contract method 0x31f09265.
//
// Solidity: function withdraw(address _to, uint256 _amount, bytes _signature) returns()
func (_IdentityImplementation *IdentityImplementationSession) Withdraw(_to common.Address, _amount *big.Int, _signature []byte) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.Withdraw(&_IdentityImplementation.TransactOpts, _to, _amount, _signature)
}

// Withdraw is a paid mutator transaction binding the contract method 0x31f09265.
//
// Solidity: function withdraw(address _to, uint256 _amount, bytes _signature) returns()
func (_IdentityImplementation *IdentityImplementationTransactorSession) Withdraw(_to common.Address, _amount *big.Int, _signature []byte) (*types.Transaction, error) {
	return _IdentityImplementation.Contract.Withdraw(&_IdentityImplementation.TransactOpts, _to, _amount, _signature)
}

// IdentityImplementationDestinationChangedIterator is returned from FilterDestinationChanged and is used to iterate over the raw logs and unpacked data for DestinationChanged events raised by the IdentityImplementation contract.
type IdentityImplementationDestinationChangedIterator struct {
	Event *IdentityImplementationDestinationChanged // Event containing the contract specifics and raw log

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
func (it *IdentityImplementationDestinationChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IdentityImplementationDestinationChanged)
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
		it.Event = new(IdentityImplementationDestinationChanged)
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
func (it *IdentityImplementationDestinationChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IdentityImplementationDestinationChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IdentityImplementationDestinationChanged represents a DestinationChanged event raised by the IdentityImplementation contract.
type IdentityImplementationDestinationChanged struct {
	PreviousDestination common.Address
	NewDestination      common.Address
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterDestinationChanged is a free log retrieval operation binding the contract event 0xe1a66d77649cf0a57b9937073549f30f1c82bb865aaf066d2f299e37a62c6aad.
//
// Solidity: event DestinationChanged(address indexed previousDestination, address indexed newDestination)
func (_IdentityImplementation *IdentityImplementationFilterer) FilterDestinationChanged(opts *bind.FilterOpts, previousDestination []common.Address, newDestination []common.Address) (*IdentityImplementationDestinationChangedIterator, error) {

	var previousDestinationRule []interface{}
	for _, previousDestinationItem := range previousDestination {
		previousDestinationRule = append(previousDestinationRule, previousDestinationItem)
	}
	var newDestinationRule []interface{}
	for _, newDestinationItem := range newDestination {
		newDestinationRule = append(newDestinationRule, newDestinationItem)
	}

	logs, sub, err := _IdentityImplementation.contract.FilterLogs(opts, "DestinationChanged", previousDestinationRule, newDestinationRule)
	if err != nil {
		return nil, err
	}
	return &IdentityImplementationDestinationChangedIterator{contract: _IdentityImplementation.contract, event: "DestinationChanged", logs: logs, sub: sub}, nil
}

// WatchDestinationChanged is a free log subscription operation binding the contract event 0xe1a66d77649cf0a57b9937073549f30f1c82bb865aaf066d2f299e37a62c6aad.
//
// Solidity: event DestinationChanged(address indexed previousDestination, address indexed newDestination)
func (_IdentityImplementation *IdentityImplementationFilterer) WatchDestinationChanged(opts *bind.WatchOpts, sink chan<- *IdentityImplementationDestinationChanged, previousDestination []common.Address, newDestination []common.Address) (event.Subscription, error) {

	var previousDestinationRule []interface{}
	for _, previousDestinationItem := range previousDestination {
		previousDestinationRule = append(previousDestinationRule, previousDestinationItem)
	}
	var newDestinationRule []interface{}
	for _, newDestinationItem := range newDestination {
		newDestinationRule = append(newDestinationRule, newDestinationItem)
	}

	logs, sub, err := _IdentityImplementation.contract.WatchLogs(opts, "DestinationChanged", previousDestinationRule, newDestinationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IdentityImplementationDestinationChanged)
				if err := _IdentityImplementation.contract.UnpackLog(event, "DestinationChanged", log); err != nil {
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

// IdentityImplementationOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the IdentityImplementation contract.
type IdentityImplementationOwnershipTransferredIterator struct {
	Event *IdentityImplementationOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *IdentityImplementationOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IdentityImplementationOwnershipTransferred)
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
		it.Event = new(IdentityImplementationOwnershipTransferred)
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
func (it *IdentityImplementationOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IdentityImplementationOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IdentityImplementationOwnershipTransferred represents a OwnershipTransferred event raised by the IdentityImplementation contract.
type IdentityImplementationOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_IdentityImplementation *IdentityImplementationFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*IdentityImplementationOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _IdentityImplementation.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &IdentityImplementationOwnershipTransferredIterator{contract: _IdentityImplementation.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_IdentityImplementation *IdentityImplementationFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *IdentityImplementationOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _IdentityImplementation.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IdentityImplementationOwnershipTransferred)
				if err := _IdentityImplementation.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// IdentityImplementationPromiseSettledIterator is returned from FilterPromiseSettled and is used to iterate over the raw logs and unpacked data for PromiseSettled events raised by the IdentityImplementation contract.
type IdentityImplementationPromiseSettledIterator struct {
	Event *IdentityImplementationPromiseSettled // Event containing the contract specifics and raw log

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
func (it *IdentityImplementationPromiseSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IdentityImplementationPromiseSettled)
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
		it.Event = new(IdentityImplementationPromiseSettled)
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
func (it *IdentityImplementationPromiseSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IdentityImplementationPromiseSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IdentityImplementationPromiseSettled represents a PromiseSettled event raised by the IdentityImplementation contract.
type IdentityImplementationPromiseSettled struct {
	Receiver     common.Address
	Amount       *big.Int
	TotalSettled *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterPromiseSettled is a free log retrieval operation binding the contract event 0x50c3491624aa1825a7653df63d067fecd5c8634ba63c99c4a7cf04ff1436070b.
//
// Solidity: event PromiseSettled(address indexed receiver, uint256 amount, uint256 totalSettled)
func (_IdentityImplementation *IdentityImplementationFilterer) FilterPromiseSettled(opts *bind.FilterOpts, receiver []common.Address) (*IdentityImplementationPromiseSettledIterator, error) {

	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _IdentityImplementation.contract.FilterLogs(opts, "PromiseSettled", receiverRule)
	if err != nil {
		return nil, err
	}
	return &IdentityImplementationPromiseSettledIterator{contract: _IdentityImplementation.contract, event: "PromiseSettled", logs: logs, sub: sub}, nil
}

// WatchPromiseSettled is a free log subscription operation binding the contract event 0x50c3491624aa1825a7653df63d067fecd5c8634ba63c99c4a7cf04ff1436070b.
//
// Solidity: event PromiseSettled(address indexed receiver, uint256 amount, uint256 totalSettled)
func (_IdentityImplementation *IdentityImplementationFilterer) WatchPromiseSettled(opts *bind.WatchOpts, sink chan<- *IdentityImplementationPromiseSettled, receiver []common.Address) (event.Subscription, error) {

	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _IdentityImplementation.contract.WatchLogs(opts, "PromiseSettled", receiverRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IdentityImplementationPromiseSettled)
				if err := _IdentityImplementation.contract.UnpackLog(event, "PromiseSettled", log); err != nil {
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

// IdentityImplementationWithdrawnIterator is returned from FilterWithdrawn and is used to iterate over the raw logs and unpacked data for Withdrawn events raised by the IdentityImplementation contract.
type IdentityImplementationWithdrawnIterator struct {
	Event *IdentityImplementationWithdrawn // Event containing the contract specifics and raw log

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
func (it *IdentityImplementationWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IdentityImplementationWithdrawn)
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
		it.Event = new(IdentityImplementationWithdrawn)
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
func (it *IdentityImplementationWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IdentityImplementationWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IdentityImplementationWithdrawn represents a Withdrawn event raised by the IdentityImplementation contract.
type IdentityImplementationWithdrawn struct {
	Receiver     common.Address
	Amount       *big.Int
	TotalBalance *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterWithdrawn is a free log retrieval operation binding the contract event 0x92ccf450a286a957af52509bc1c9939d1a6a481783e142e41e2499f0bb66ebc6.
//
// Solidity: event Withdrawn(address indexed receiver, uint256 amount, uint256 totalBalance)
func (_IdentityImplementation *IdentityImplementationFilterer) FilterWithdrawn(opts *bind.FilterOpts, receiver []common.Address) (*IdentityImplementationWithdrawnIterator, error) {

	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _IdentityImplementation.contract.FilterLogs(opts, "Withdrawn", receiverRule)
	if err != nil {
		return nil, err
	}
	return &IdentityImplementationWithdrawnIterator{contract: _IdentityImplementation.contract, event: "Withdrawn", logs: logs, sub: sub}, nil
}

// WatchWithdrawn is a free log subscription operation binding the contract event 0x92ccf450a286a957af52509bc1c9939d1a6a481783e142e41e2499f0bb66ebc6.
//
// Solidity: event Withdrawn(address indexed receiver, uint256 amount, uint256 totalBalance)
func (_IdentityImplementation *IdentityImplementationFilterer) WatchWithdrawn(opts *bind.WatchOpts, sink chan<- *IdentityImplementationWithdrawn, receiver []common.Address) (event.Subscription, error) {

	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _IdentityImplementation.contract.WatchLogs(opts, "Withdrawn", receiverRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IdentityImplementationWithdrawn)
				if err := _IdentityImplementation.contract.UnpackLog(event, "Withdrawn", log); err != nil {
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
