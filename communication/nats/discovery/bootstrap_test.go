package discovery

import (
	"encoding/json"
	"github.com/mysterium/node/openvpn"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func init() {
	openvpn.Bootstrap()
	Bootstrap()
}

func TestServiceProposalUnserializeNatsContact(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "openvpn",
		"service_definition": {},
		"payment_method_type": "PER_TIME",
		"payment_method": {},
		"provider_contacts": [
			{
				"type": "nats/v1",
				"definition": {
					"topic": "test-topic"
				}
			}
		]
	}`)

	var actual dto_discovery.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.Nil(t, err)
	assert.Len(t, actual.ProviderContacts, 1)
	assert.Exactly(
		t,
		dto_discovery.Contact{
			Type: TypeContactNATSV1,
			Definition: ContactNATSV1{
				Topic: "test-topic",
			},
		},
		actual.ProviderContacts[0],
	)
}
