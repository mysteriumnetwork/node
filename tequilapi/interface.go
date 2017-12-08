package tequilapi

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type CollectionTequilaInterface interface {
	Get(http.ResponseWriter, *http.Request, httprouter.Params)
}
