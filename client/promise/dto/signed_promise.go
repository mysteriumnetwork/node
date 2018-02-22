package dto

// SignedPromise represents payment promise signed by issuer
type SignedPromise struct {
	Promise         PromiseBody
	IssuerSignature Signature
}
