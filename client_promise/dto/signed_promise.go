package dto

type SignedPromise struct {
	Promise         PromiseBody
	IssuerSignature Signature
}
