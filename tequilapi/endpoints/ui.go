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

package endpoints

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/ui/versionmanager"
)

// NodeUIEndpoints node ui version management endpoints
type NodeUIEndpoints struct {
	versionManager *versionmanager.VersionManager
}

// NewNodeUIEndpoints constructor
func NewNodeUIEndpoints(versionManager *versionmanager.VersionManager) *NodeUIEndpoints {
	return &NodeUIEndpoints{
		versionManager: versionManager,
	}
}

// RemoteVersions list node UI releases from repo
// swagger:operation GET /ui/remote-versions UI uiRemoteVersions
//
//	---
//	summary: List local
//	description: provides a list of node UI releases from github repository
//	responses:
//	  200:
//	    description: Remote version list
//	    schema:
//	      "$ref": "#/definitions/RemoteVersionsResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (n *NodeUIEndpoints) RemoteVersions(c *gin.Context) {
	r := versionmanager.RemoteVersionRequest{
		PerPage: 50,
	}

	if b, err := strconv.ParseBool(c.Query("flush_cache")); err == nil {
		r.FlushCache = b
	}

	if pp, err := strconv.ParseInt(c.Query("per_page"), 10, 32); err == nil {
		r.PerPage = int(pp)
	}

	if p, err := strconv.ParseInt(c.Query("page"), 10, 32); err == nil {
		r.Page = int(p)
	}

	versions, err := n.versionManager.ListRemoteVersions(r)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, contract.RemoteVersionsResponse{Versions: versions})
}

// LocalVersions locally available node UI versions
// swagger:operation GET /ui/local-versions UI uiLocalVersions
//
//	---
//	summary: List remote
//	description: provides a list of node UI releases that have been downloaded or bundled with node
//	responses:
//	  200:
//	    description: Local version list
//	    schema:
//	      "$ref": "#/definitions/LocalVersionsResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (n *NodeUIEndpoints) LocalVersions(c *gin.Context) {
	versions, err := n.versionManager.ListLocalVersions()
	if err != nil {
		c.Error(apierror.Internal("Could not list local versions: "+err.Error(), contract.ErrCodeUILocalVersions))
		return
	}
	c.JSON(http.StatusOK, contract.LocalVersionsResponse{Versions: versions})
}

// SwitchVersion switches node UI version to locally available one
// swagger:operation POST /ui/switch-version UI uiSwitchVersion
//
//	---
//	summary: Switch Version
//	description: switch node UI version to locally available one
//	responses:
//	  200:
//	    description: version switched
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  422:
//	    description: Unable to process the request at this point
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (n *NodeUIEndpoints) SwitchVersion(c *gin.Context) {
	var req contract.SwitchNodeUIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	if err := req.Valid(); err != nil {
		c.Error(err)
		return
	}

	if err := n.versionManager.SwitchTo(req.Version); err != nil {
		c.Error(apierror.Unprocessable(fmt.Sprintf("Could not switch to node UI version: %s", req.Version), contract.ErrCodeUISwitchVersion))
		return
	}

	c.AbortWithStatus(http.StatusOK)
}

// Download download a remote node UI release
// swagger:operation POST /ui/download-version UI uiDownload
//
//	---
//	summary: Download
//	description: download a remote node UI release
//	responses:
//	  200:
//	    description: Download in progress
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (n *NodeUIEndpoints) Download(c *gin.Context) {
	var req contract.DownloadNodeUIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	if err := req.Valid(); err != nil {
		c.Error(err)
		return
	}

	if err := n.versionManager.Download(req.Version); err != nil {
		c.Error(apierror.Internal(fmt.Sprintf("Could not download node UI version: %s", req.Version), contract.ErrCodeUIDownload))
		return
	}

	c.AbortWithStatus(http.StatusOK)
}

// DownloadStatus returns download status
// swagger:operation GET /ui/download-status UI uiDownloadStatus
//
//	---
//	summary: Download status
//	description: DownloadStatus can download one remote release at a time. This endpoint provides status of the download.
//	responses:
//	  200:
//	    description: download status
//	    schema:
//	      "$ref": "#/definitions/DownloadStatus"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (n *NodeUIEndpoints) DownloadStatus(c *gin.Context) {
	c.JSON(http.StatusOK, n.versionManager.DownloadStatus())
}

// UI returns download status
// swagger:operation GET /ui/info UI ui
//
//	---
//	summary: Node UI information
//	description: Node UI information
//	responses:
//	  200:
//	    description: Node UI information
//	    schema:
//	      "$ref": "#/definitions/UI"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (n *NodeUIEndpoints) UI(c *gin.Context) {
	bundled, err := n.versionManager.BundledVersion()
	if err != nil {
		c.Error(apierror.Internal("Could not resolve bundled version: "+err.Error(), contract.ErrCodeUIBundledVersion))
		return
	}

	used, err := n.versionManager.UsedVersion()
	if err != nil {
		c.Error(apierror.Internal("Could not resolve used version: "+err.Error(), contract.ErrCodeUIUsedVersion))
		return
	}
	c.JSON(http.StatusOK, contract.UI{
		BundledVersion: bundled.Name,
		UsedVersion:    used.Name,
	})
}

// AddRoutesForNodeUI provides controls for nodeUI management via tequilapi
func AddRoutesForNodeUI(versionManager *versionmanager.VersionManager) func(*gin.Engine) error {
	endpoints := NewNodeUIEndpoints(versionManager)

	return func(e *gin.Engine) error {
		v1Group := e.Group("/ui")
		{
			v1Group.GET("/info", endpoints.UI)
			v1Group.GET("/local-versions", endpoints.LocalVersions)
			v1Group.GET("/remote-versions", endpoints.RemoteVersions)
			v1Group.POST("/switch-version", endpoints.SwitchVersion)
			v1Group.POST("/download-version", endpoints.Download)
			v1Group.GET("/download-status", endpoints.DownloadStatus)
		}
		return nil
	}
}
