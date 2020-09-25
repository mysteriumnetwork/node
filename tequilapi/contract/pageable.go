/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package contract

import (
	"net/http"

	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

// NewPaginationQuery creates pagination query from API request.
func NewPaginationQuery(request *http.Request) (PaginationQuery, *validation.FieldErrorMap) {
	query := request.URL.Query()
	errs := validation.NewErrorMap()

	return PaginationQuery{
		PageSize: parseIntOptional(query.Get("page_size"), errs.ForField("page_size")),
		Page:     parseIntOptional(query.Get("page"), errs.ForField("page")),
	}, errs
}

// PaginationQuery allows to page response items.
type PaginationQuery struct {
	// Number of items per page.
	// in: query
	// default: 50
	PageSize *int `json:"page_size"`

	// Page to filter the items by.
	// in: query
	// default: 1
	Page *int `json:"page"`
}

// NewPageableDTO maps to API pagination DTO.
func NewPageableDTO(paginator *utils.Paginator) PageableDTO {
	return PageableDTO{
		Page:       paginator.Page(),
		PageSize:   paginator.PageSize(),
		TotalItems: paginator.Nums(),
		TotalPages: paginator.PageNums(),
	}
}

// PageableDTO holds pagination information.
// swagger:model PageableDTO
type PageableDTO struct {
	// The current page of the items.
	Page int `json:"page"`
	// Number of items per page.
	PageSize int `json:"page_size"`
	// The total items.
	TotalItems int `json:"total_items"`
	// The last page of the items.
	TotalPages int `json:"total_pages"`
}
