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

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/feedback"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/pkg/errors"
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
	IssueID string `json:"issueId"`
}

// ReportIssueError issue report error
// swagger:model
type ReportIssueError struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// ReportIssue reports user issue
// swagger:operation POST /feedback/issue Feedback reportIssue
// ---
// summary: Reports user issue
// description: Reports user issue
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
func (api *feedbackAPI) ReportIssue(httpRes http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	req := feedback.UserReport{}
	err := json.NewDecoder(httpReq.Body).Decode(&req)
	if err != nil {
		utils.SendError(httpRes, errors.Wrap(err, "could not read message body"), http.StatusBadRequest)
		return
	}

	result, err := api.reporter.NewIssue(req)
	if err != nil {
		utils.SendError(httpRes, err, http.StatusInternalServerError)
		return
	}

	if !result.Success {
		utils.SendErrorBody(httpRes, result.Errors, result.HTTPResponse.StatusCode)
		return
	}

	utils.WriteAsJSON(result.Response, httpRes)
}

// AddRoutesForFeedback registers feedback routes
func AddRoutesForFeedback(
	router *httprouter.Router,
	reporter *feedback.Reporter,
) {
	api := newFeedbackAPI(reporter)
	router.POST("/feedback/issue", api.ReportIssue)
}
