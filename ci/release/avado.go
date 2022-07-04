/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package release

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/mysteriumnetwork/go-ci/job"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/node/logconfig"
)

type avadoPR struct {
	client *github.Client

	version string

	avadoUser     string
	mysteriumUser string

	msg    string
	branch string

	repo string

	authorName  string
	authorEmail string
}

func newAvadoPR(ctx context.Context) *avadoPR {
	token := env.Str(env.GithubAPIToken)
	version := env.Str(env.BuildVersion)
	owner := env.Str(env.GithubOwner)

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &avadoPR{
		version: version,
		client:  client,

		avadoUser:     "AvadoDServer",
		mysteriumUser: owner,

		repo: "AVADO-DNP-Mysterium-Server",

		msg:    "Update Mysterium Node version to " + version,
		branch: "update-mysterium-node-" + version,

		authorName:  "MysteriumTeam",
		authorEmail: "core-services@mysterium.network",
	}
}

func (a *avadoPR) getRef(ctx context.Context) (ref *github.Reference, err error) {
	var baseRef *github.Reference
	if baseRef, _, err = a.client.Git.GetRef(ctx, a.avadoUser, a.repo, "refs/heads/master"); err != nil {
		return nil, err
	}
	newRef := &github.Reference{Ref: github.String("refs/heads/" + a.branch), Object: &github.GitObject{SHA: baseRef.Object.SHA}}
	ref, _, err = a.client.Git.CreateRef(ctx, a.mysteriumUser, a.repo, newRef)
	return ref, err
}

func (a *avadoPR) getTree(ctx context.Context, ref *github.Reference) (tree *github.Tree, err error) {
	entries := []*github.TreeEntry{}

	for _, fileName := range []string{"dappnode_package.json", "build/Dockerfile"} {
		file, content, err := a.getFileContent(ctx, fileName)
		if err != nil {
			return nil, err
		}
		content = replaceVersion(file, content, a.version)
		entries = append(entries, &github.TreeEntry{Path: github.String(file), Type: github.String("blob"), Content: github.String(content), Mode: github.String("100644")})
	}

	tree, _, err = a.client.Git.CreateTree(ctx, a.mysteriumUser, a.repo, *ref.Object.SHA, entries)
	return tree, err
}

func (a *avadoPR) getFileContent(ctx context.Context, fileArg string) (targetName string, s string, err error) {
	f, _, _, err := a.client.Repositories.GetContents(ctx, a.avadoUser, a.repo, fileArg, nil)
	if err != nil {
		return "", "", err
	}

	c, err := f.GetContent()
	return fileArg, c, err
}

func (a *avadoPR) pushCommit(ctx context.Context, ref *github.Reference, tree *github.Tree) (err error) {
	parent, _, err := a.client.Repositories.GetCommit(ctx, a.mysteriumUser, a.repo, *ref.Object.SHA)
	if err != nil {
		return err
	}

	parent.Commit.SHA = parent.SHA

	date := time.Now()
	author := &github.CommitAuthor{Date: &date, Name: &a.authorName, Email: &a.authorEmail}
	commit := &github.Commit{Author: author, Message: &a.msg, Tree: tree, Parents: []*github.Commit{parent.Commit}}
	newCommit, _, err := a.client.Git.CreateCommit(ctx, a.mysteriumUser, a.repo, commit)
	if err != nil {
		return err
	}

	ref.Object.SHA = newCommit.SHA
	_, _, err = a.client.Git.UpdateRef(ctx, a.mysteriumUser, a.repo, ref, false)
	return err
}

func (a *avadoPR) createPR(ctx context.Context) (err error) {
	base := "master"
	head := fmt.Sprintf("%s:%s", a.mysteriumUser, a.branch)

	newPR := &github.NewPullRequest{
		Title:               &a.msg,
		Head:                &head,
		Base:                &base,
		MaintainerCanModify: github.Bool(true),
	}

	_, _, err = a.client.PullRequests.Create(ctx, a.avadoUser, a.repo, newPR)
	if err != nil {
		return err
	}

	return nil
}

func replaceVersion(file, content, version string) (out string) {
	oldLine := ""
	newLine := ""
	switch file {
	case "dappnode_package.json":
		oldLine = `  "upstream": `
		newLine = fmt.Sprintf(`  "upstream": "%s",`, version)
	case "build/Dockerfile":
		oldLine = "FROM mysteriumnetwork/myst:"
		newLine = fmt.Sprintf(`FROM mysteriumnetwork/myst:%s-alpine`, version)
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, oldLine) && !strings.HasPrefix(line, "#") {
			out += newLine + "\n"
		} else {
			out += line + "\n"
		}
	}

	return out
}

// CreateAvadoPR create github PR to the Avado repository.
func CreateAvadoPR() error {
	logconfig.Bootstrap()

	if err := env.EnsureEnvVars(
		env.GithubAPIToken,
		env.GithubOwner,
		env.BuildVersion,
	); err != nil {
		return err
	}
	job.Precondition(func() bool {
		return env.Bool(env.TagBuild) && !env.Bool(env.RCBuild)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	a := newAvadoPR(ctx)

	ref, err := a.getRef(ctx)
	if err != nil || ref == nil {
		log.Error().Msgf("Unable to get/create the commit reference: %s\n", err)
		return err
	}

	tree, err := a.getTree(ctx, ref)
	if err != nil {
		log.Error().Msgf("Unable to create the tree based on the provided files: %s\n", err)
		return err
	}

	if err := a.pushCommit(ctx, ref, tree); err != nil {
		log.Error().Msgf("Unable to create the commit: %s\n", err)
		return err
	}

	if err := a.createPR(ctx); err != nil {
		log.Error().Msgf("Error while creating the pull request: %s", err)
		return err
	}

	return nil
}
