package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
)

type currentIPEndpoint struct {
	ipResolver ipResolver
}

type ipResolver func() (string, error)

// NewCurrentIPEndpoint returns endpoint for detecting current ip
func NewCurrentIPEndpoint(ipResolver ipResolver) *currentIPEndpoint {
	return &currentIPEndpoint{
		ipResolver: ipResolver,
	}
}

// CurrentIP responds with current ip, using its ip resolver
func (ipe *currentIPEndpoint) CurrentIP(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ip, err := ipe.ipResolver()
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}
	response := struct {
		IP string `json:"ip"`
	}{
		IP: ip,
	}
	utils.WriteAsJSON(response, writer)
}
