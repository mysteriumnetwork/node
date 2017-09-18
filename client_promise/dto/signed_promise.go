package dto

type SignedPromise struct {
	promise         PromiseBody
	issuerSignature Signature
}