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

package endpoints

import (
	"math/big"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/go-rest/apierror"

	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// Affiliator represents interface to affiliator service
type Affiliator interface {
	RegistrationTokenReward(token string) (*big.Int, error)
}

type affiliatorEndpoint struct {
	affiliator Affiliator
}

// NewAffiliatorEndpoint creates and returns affiliator endpoint
func NewAffiliatorEndpoint(affiliator Affiliator) *affiliatorEndpoint {
	return &affiliatorEndpoint{affiliator: affiliator}
}

// swagger:operation POST /affiliator/token/{token}/reward AffiliatorTokenReward
//
//	---
//	summary: Returns the amount of reward for a token (affiliator)
//	parameters:
//	- in: path
//	  name: token
//	  description: Token for which to lookup the reward
//	  type: string
//	  required: true
//	responses:
//	  200:
//	    description: Token Reward
//	    schema:
//	      "$ref": "#/definitions/TokenRewardAmount"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (a *affiliatorEndpoint) TokenRewardAmount(c *gin.Context) {
	token := c.Param("token")
	reward, err := a.affiliator.RegistrationTokenReward(token)
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Could not fetch reward", contract.ErrCodeAffiliatorFailed))
		return
	}
	if reward == nil {
		c.Error(apierror.Internal("No reward for token", contract.ErrCodeAffiliatorNoReward))
		return
	}

	utils.WriteAsJSON(contract.TokenRewardAmount{
		Amount: reward,
	}, c.Writer)
}

// AddRoutesForAffiliator attaches Affiliator endpoints to router
func AddRoutesForAffiliator(affiliator Affiliator) func(*gin.Engine) error {
	a := NewAffiliatorEndpoint(affiliator)

	return func(e *gin.Engine) error {
		affiliatorGroup := e.Group("/affiliator")
		{
			affiliatorGroup.GET("/token/:token/reward", a.TokenRewardAmount)
		}
		return nil
	}
}
