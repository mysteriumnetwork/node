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
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/feedback"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type feedbackAPI struct {
	reporter *feedback.Reporter
}

func newFeedbackAPI(reporter *feedback.Reporter) *feedbackAPI {
	return &feedbackAPI{reporter: reporter}
}

// ReportIssueRequest params for issue report
// swagger:model
type ReportIssueRequest struct {
	Email       string `json:"email"`
	Description string `json:"description"`
}

// ReportIssueSuccess successful issue report
// swagger:model
type ReportIssueSuccess struct {
	IssueID string `json:"issue_id"`
}

// ReportIssueError issue report error
// swagger:model
type ReportIssueError struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// ReportIssueGithub reports user issue to github
// swagger:operation POST /feedback/issue Feedback reportIssueGithub
// ---
// summary: Reports user issue to github
// description: Reports user issue to github
// parameters:
//   - in: body
//     name: body
//     description: Report issue request
//     schema:
//       $ref: "#/definitions/ReportIssueRequest"
// responses:
//   200:
//     description: Issue reported
//     schema:
//       "$ref": "#/definitions/ReportIssueSuccess"
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ReportIssueError"
//   429:
//     description: Too many requests (max. 1/minute)
//     schema:
//       "$ref": "#/definitions/ReportIssueError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ReportIssueError"
func (api *feedbackAPI) ReportIssueGithub(c *gin.Context) {
	httpReq := c.Request
	httpRes := c.Writer

	req := feedback.UserReport{}
	err := json.NewDecoder(httpReq.Body).Decode(&req)
	if err != nil {
		utils.SendError(httpRes, errors.Wrap(err, "could not read message body"), http.StatusBadRequest)
		return
	}

	result, err := api.reporter.NewIssue(req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Could not create an issue for feedback")
		utils.SendError(httpRes, err, http.StatusInternalServerError)
		return
	}

	if !result.Success {
		log.Error().Stack().Err(err).Msg("Submitting an issue failed")
		utils.WriteAsJSON(result.Errors, httpRes, result.HTTPResponse.StatusCode)
		return
	}

	utils.WriteAsJSON(result.Response, httpRes)
}

// ReportIntercomIssueRequest params for intercom issue report
// swagger:model
type ReportIntercomIssueRequest struct {
	Email       string `json:"email"`
	Description string `json:"description"`
	UserId      string `json:"user_id"`
	UserType    string `json:"user_type"`
}

// ReportIssueIntercom reports user issue to intercom
// swagger:operation POST /feedback/issue/intercom Feedback reportIssueIntercom
// ---
// summary: Reports user issue to intercom
// description: Reports user user to intercom
// parameters:
//   - in: body
//     name: body
//     description: Report issue request
//     schema:
//       $ref: "#/definitions/ReportIntercomIssueRequest"
// responses:
//   201:
//     description: Issue reported
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ReportIssueError"
//   429:
//     description: Too many requests (max. 1/minute)
//     schema:
//       "$ref": "#/definitions/ReportIssueError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ReportIssueError"
func (api *feedbackAPI) ReportIssueIntercom(c *gin.Context) {
	httpReq := c.Request
	httpRes := c.Writer

	req := feedback.UserReport{}
	err := json.NewDecoder(httpReq.Body).Decode(&req)
	if err != nil {
		utils.SendError(httpRes, errors.Wrap(err, "could not read message body"), http.StatusBadRequest)
		return
	}

	result, err := api.reporter.NewIntercomIssue(req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Could not create an issue for feedback")
		utils.SendError(httpRes, err, http.StatusInternalServerError)
		return
	}

	if !result.Success {
		log.Error().Stack().Err(err).Msg("Submitting an issue failed")
		utils.WriteAsJSON(result.Errors, httpRes, result.HTTPResponse.StatusCode)
		return
	}
}

// AddRoutesForFeedback registers feedback routes
func AddRoutesForFeedback(
	reporter *feedback.Reporter,
) func(*gin.Engine) error {
	api := newFeedbackAPI(reporter)
	return func(g *gin.Engine) error {
		g.POST("/feedback/issue", api.ReportIssueGithub)
		g.POST("/feedback/issue/intercom", api.ReportIssueIntercom)
		return nil
	}
}
