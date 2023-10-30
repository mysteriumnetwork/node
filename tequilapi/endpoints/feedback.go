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
	"github.com/mysteriumnetwork/go-rest/apierror"

	"github.com/mysteriumnetwork/node/feedback"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/rs/zerolog/log"
)

type feedbackAPI struct {
	reporter *feedback.Reporter
}

func newFeedbackAPI(reporter *feedback.Reporter) *feedbackAPI {
	return &feedbackAPI{reporter: reporter}
}

// ReportIssueGithubResponse response for github issue creation
//
// swagger:model ReportIssueGithubResponse
type ReportIssueGithubResponse struct {
	IssueID string `json:"issue_id"`
}

// ReportIssueGithub reports user issue to github
// swagger:operation POST /feedback/issue Feedback reportIssueGithub
//
//	---
//	summary: Reports user issue to github
//	description: Reports user issue to github
//	deprecated: true
//	parameters:
//	  - in: body
//	    name: body
//	    description: Bug report issue request
//	    schema:
//	      $ref: "#/definitions/BugReport"
//	responses:
//	  200:
//	    description: Issue reported
//	    schema:
//	      "$ref": "#/definitions/ReportIssueGithubResponse"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  429:
//	    description: Too many requests (max. 1/minute)
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  503:
//	    description: Unavailable
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *feedbackAPI) ReportIssueGithub(c *gin.Context) {
	var req feedback.BugReport
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	result, apiErr, err := api.reporter.NewIssue(req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Could not create an issue for feedback")
	}
	if apiErr != nil {
		c.Error(apiErr)
		return
	} else if err != nil {
		c.Error(err)
		return
	}

	utils.WriteAsJSON(ReportIssueGithubResponse{
		IssueID: result.IssueId,
	}, c.Writer)
}

// ReportIssueIntercom reports user issue to intercom
// swagger:operation POST /feedback/issue/intercom Feedback reportIssueIntercom
//
//	---
//	summary: Reports user issue to intercom
//	description: Reports user user to intercom
//	deprecated: true
//	parameters:
//	  - in: body
//	    name: body
//	    description: Report issue request
//	    schema:
//	      $ref: "#/definitions/UserReport"
//	responses:
//	  201:
//	    description: Issue reported
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  429:
//	    description: Too many requests (max. 1/minute)
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  503:
//	    description: Unavailable
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *feedbackAPI) ReportIssueIntercom(c *gin.Context) {
	var req feedback.UserReport
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	_, apiErr, err := api.reporter.NewIntercomIssue(req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Could not create an issue for feedback")
	}
	if apiErr != nil {
		c.Error(apiErr)
		return
	} else if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusCreated)
}

// BugReport reports a bug with logs
// swagger:operation POST /feedback/bug-report Feedback bugReport
//
//	---
//	summary: Creates a bug report
//	description: Creates a bug report with logs
//	parameters:
//	  - in: body
//	    name: body
//	    description: Report a bug
//	    schema:
//	      $ref: "#/definitions/BugReport"
//	responses:
//	  200:
//	    description: Bug report response
//	    schema:
//	      "$ref": "#/definitions/CreateBugReportResponse"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  429:
//	    description: Too many requests (max. 1/minute)
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  503:
//	    description: Unavailable
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *feedbackAPI) BugReport(c *gin.Context) {
	var req feedback.BugReport
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	result, apiErr, err := api.reporter.NewBugReport(req)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Could not create a bug report")
	}
	if apiErr != nil {
		c.Error(apiErr)
		return
	} else if err != nil {
		c.Error(err)
		return
	}

	utils.WriteAsJSON(*result, c.Writer)
}

// AddRoutesForFeedback registers feedback routes
func AddRoutesForFeedback(
	reporter *feedback.Reporter,
) func(*gin.Engine) error {
	api := newFeedbackAPI(reporter)
	return func(g *gin.Engine) error {
		g.POST("/feedback/issue", api.ReportIssueGithub)
		g.POST("/feedback/issue/intercom", api.ReportIssueIntercom)
		g.POST("/feedback/bug-report", api.BugReport)
		return nil
	}
}
