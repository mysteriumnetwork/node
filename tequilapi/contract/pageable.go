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

const (
	defaultPageSize = 50
	defaultPage     = 1
)

// NewPaginationQuery creates pagination query with default values.
func NewPaginationQuery() PaginationQuery {
	return PaginationQuery{
		PageSize: defaultPageSize,
		Page:     defaultPage,
	}
}

// PaginationQuery allows to page response items.
type PaginationQuery struct {
	// Number of items per page.
	// in: query
	// default: 50
	PageSize int `json:"page_size"`

	// Page to filter the items by.
	// in: query
	// default: 1
	Page int `json:"page"`
}

// Bind creates and validates query from API request.
func (q *PaginationQuery) Bind(request *http.Request) *validation.FieldErrorMap {
	errs := validation.NewErrorMap()

	qs := request.URL.Query()
	if qStr := qs.Get("page_size"); qStr != "" {
		if qVal, err := parseInt(qStr); err != nil {
			errs.ForField("page_size").Add(err)
		} else {
			q.PageSize = *qVal
		}
	}
	if qStr := qs.Get("page"); qStr != "" {
		if qVal, err := parseInt(qStr); err != nil {
			errs.ForField("page").Add(err)
		} else {
			q.Page = *qVal
		}
	}

	return errs
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
