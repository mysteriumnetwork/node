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

package feedback

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/mysteriumnetwork/feedback/client"
	"github.com/mysteriumnetwork/feedback/feedback"
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/rs/zerolog/log"
)

// Reporter reports issues from users
type Reporter struct {
	logCollector     logCollector
	identityProvider identityProvider
	feedbackAPI      *client.FeedbackAPI
	originResolver   location.OriginResolver
}

// NewReporter constructs a new Reporter
func NewReporter(
	logCollector logCollector,
	identityProvider identityProvider,
	originResolver location.OriginResolver,
	feedbackURL string,
) (*Reporter, error) {
	log.Info().Msg("Using feedback API at: " + feedbackURL)
	api, err := client.NewFeedbackAPI(feedbackURL)
	if err != nil {
		return nil, err
	}
	return &Reporter{
		logCollector:     logCollector,
		identityProvider: identityProvider,
		originResolver:   originResolver,
		feedbackAPI:      api,
	}, nil
}

type logCollector interface {
	Archive() (filepath string, err error)
}

type identityProvider interface {
	GetIdentities() []identity.Identity
}

// BugReport represents user input when submitting an issue report
// swagger:model BugReport
type BugReport struct {
	Email       string `json:"email"`
	Description string `json:"description"`
}

// Validate validates a bug report
func (br *BugReport) Validate() *apierror.APIError {
	v := apierror.NewValidator()
	br.Email = strings.TrimSpace(br.Email)
	if br.Email == "" {
		v.Required("email")
	} else if _, err := mail.ParseAddress(br.Email); err != nil {
		v.Invalid("email", "Invalid email address")
	}

	br.Description = strings.TrimSpace(br.Description)
	if len(br.Description) < 30 {
		v.Invalid("description", "Description too short. Provide at least 30 character long description.")
	}

	return v.Err()
}

// NewIssue sends node logs, Identity and UserReport to the feedback service
func (r *Reporter) NewIssue(report BugReport) (*feedback.CreateGithubIssueResponse, *apierror.APIError, error) {
	if apiErr := report.Validate(); apiErr != nil {
		return nil, apiErr, fmt.Errorf("invalid report: %w", apiErr)
	}

	userID := r.currentIdentity()

	archiveFilepath, err := r.logCollector.Archive()
	if err != nil {
		return nil, apierror.Internal("could not create log archive", "cannot_get_logs"), fmt.Errorf("could not create log archive: %w", err)
	}

	result, apierr, err := r.feedbackAPI.CreateGithubIssue(feedback.CreateGithubIssueRequest{
		UserId:      userID,
		Description: report.Description,
		Email:       report.Email,
	}, archiveFilepath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create github issue: %w", err)
	}

	return result, apierr, nil
}

// UserReport represents user input when submitting an issue report
// swagger:model UserReport
type UserReport struct {
	BugReport
	UserId   string `json:"user_id"`
	UserType string `json:"user_type"`
}

// Validate validate UserReport
func (ur *UserReport) Validate() *apierror.APIError {
	return ur.BugReport.Validate()
}

// NewIntercomIssue sends node logs, Identity and UserReport to intercom
func (r *Reporter) NewIntercomIssue(report UserReport) (*feedback.CreateIntercomIssueResponse, *apierror.APIError, error) {
	if apiErr := report.Validate(); apiErr != nil {
		return nil, apiErr, fmt.Errorf("invalid report: %w", apiErr)
	}

	nodeID := r.currentIdentity()
	location := r.originResolver.GetOrigin()

	archiveFilepath, err := r.logCollector.Archive()
	if err != nil {
		return nil, apierror.Internal("could not create log archive", "cannot_get_logs"), fmt.Errorf("could not create log archive: %w", err)
	}

	result, apierr, err := r.feedbackAPI.CreateIntercomIssue(feedback.CreateIntercomIssueRequest{
		UserId:       report.UserId,
		Description:  report.Description,
		Email:        report.Email,
		NodeIdentity: nodeID,
		NodeCountry:  location.Country,
		IpType:       location.IPType,
		Ip:           location.IP,
		UserType:     report.UserType,
	}, archiveFilepath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create intercom issue: %w", err)
	}

	return result, apierr, nil
}

// CreateBugReportResponse response for bug report creation
// swagger:model CreateBugReportResponse
type CreateBugReportResponse struct {
	Message     string `json:"message"`
	Email       string `json:"email"`
	Identity    string `json:"identity"`
	NodeCountry string `json:"node_country"`
	IpType      string `json:"ip_type"`
	Ip          string `json:"ip"`
}

// NewBugReport creates a new bug report and returns the message that can be sent to intercom
func (r *Reporter) NewBugReport(report BugReport) (*CreateBugReportResponse, *apierror.APIError, error) {
	if apiErr := report.Validate(); apiErr != nil {
		return nil, apiErr, fmt.Errorf("invalid report: %w", apiErr)
	}

	nodeID := r.currentIdentity()
	location := r.originResolver.GetOrigin()

	archiveFilepath, err := r.logCollector.Archive()
	if err != nil {
		return nil, apierror.Internal("could not create log archive", "cannot_get_logs"), fmt.Errorf("could not create log archive: %w", err)
	}

	result, apierr, err := r.feedbackAPI.CreateBugReport(feedback.CreateBugReportRequest{
		NodeIdentity: nodeID,
		Description:  report.Description,
		Email:        report.Email,
	}, archiveFilepath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create intercom issue: %w", err)
	} else if apierr != nil {
		return nil, apierr, apierr
	}

	return &CreateBugReportResponse{
		Message:     result.Message,
		Email:       result.Email,
		Identity:    result.NodeIdentity,
		NodeCountry: location.Country,
		IpType:      location.IPType,
		Ip:          location.IP,
	}, nil, nil
}

func (r *Reporter) currentIdentity() (identity string) {
	identities := r.identityProvider.GetIdentities()
	if len(identities) > 0 {
		return identities[0].Address
	}
	return "unknown_identity"
}
