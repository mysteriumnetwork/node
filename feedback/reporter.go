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
	"github.com/mysteriumnetwork/feedback/client"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Reporter reports issues from users
type Reporter struct {
	logCollector     logCollector
	identityProvider identityProvider
	feedbackAPI      *client.FeedbackAPI
}

// NewReporter constructs a new Reporter
func NewReporter(
	logCollector logCollector,
	identityProvider identityProvider,
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
		feedbackAPI:      api,
	}, nil
}

type logCollector interface {
	Archive() (filepath string, err error)
}

type identityProvider interface {
	GetIdentities() []identity.Identity
}

// UserReport represents user input when submitting an issue report
type UserReport struct {
	Email       string `json:"email"`
	Description string `json:"description"`
}

// NewIssue sends node logs, Identity and UserReport to the feedback service
func (r *Reporter) NewIssue(report UserReport) (result *client.CreateGithubIssueResult, err error) {
	userID := r.currentIdentity()

	archiveFilepath, err := r.logCollector.Archive()
	if err != nil {
		return nil, errors.Wrap(err, "could not create log archive")
	}

	result, err = r.feedbackAPI.CreateGithubIssue(client.CreateGithubIssueRequest{
		UserId:      userID,
		Description: report.Description,
		Email:       report.Email,
		Filepath:    archiveFilepath,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not create github issue")
	}
	return result, nil
}

func (r *Reporter) currentIdentity() (identity string) {
	identities := r.identityProvider.GetIdentities()
	if len(identities) > 0 {
		return identities[0].Address
	}
	return "unknown_identity"
}
