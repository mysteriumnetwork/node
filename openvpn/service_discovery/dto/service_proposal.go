package dto

import (
	"github.com/mysterium/node/service_discovery/dto"
	"encoding/json"
	"log"
)

type proposal struct {
	Id                int              `json:"id"`
	Format            string           `json:"format"`
	ServiceType       string           `json:"service_type"`
	ProviderId        string           `json:"provider_id"`
	PaymentMethodType string           `json:"payment_method_type"`
	ServiceDefinition *json.RawMessage `json:"service_definition"`
	PaymentMethod     *json.RawMessage `json:"payment_method"`
	ProviderContacts  []dto.Contact    `json:"provider_contacts"`
}

func BuildServiceProposalFromJson(jsonData []byte) dto.ServiceProposal {
	proposal := proposal{}

	if err := json.Unmarshal([]byte(jsonData), &proposal); err != nil {
		log.Fatal(err)
	}

	ovProposal := dto.ServiceProposal{
		Id:                proposal.Id,
		Format:            proposal.Format,
		ServiceType:       proposal.ServiceType,
		ProviderId:        dto.Identity(proposal.ProviderId),
		PaymentMethodType: proposal.PaymentMethodType,
		ProviderContacts:  proposal.ProviderContacts,
	}

	switch proposal.ServiceType {
	case "openvpn":
		ovProposal.ServiceDefinition = getServiceDefinitionFromJson(proposal.ServiceDefinition)
	}

	ovProposal.PaymentMethod = getPaymentMethodFromJson(proposal.PaymentMethodType, proposal.PaymentMethod)

	return ovProposal
}

func getServiceDefinitionFromJson(rawDefinition *json.RawMessage) ServiceDefinition {
	definition := ServiceDefinition{}

	if err := json.Unmarshal(*rawDefinition, &definition); err != nil {
		log.Fatal(err)
	}

	return definition
}

func getPaymentMethodFromJson(paymentMethodType string, rawMethod *json.RawMessage) dto.PaymentMethod {
	switch paymentMethodType {
	case PAYMENT_METHOD_PER_TIME:
		var method PaymentMethodPerTime

		if err := json.Unmarshal(*rawMethod, &method); err != nil {
			log.Fatal(err)
		}

		return method

	case PAYMENT_METHOD_PER_BYTES:
		var method PaymentMethodPerBytes

		if err := json.Unmarshal(*rawMethod, &method); err != nil {
			log.Fatal(err)
		}

		return method
	}

	return nil
}
