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
	"github.com/vcraescu/go-paginator"
)

// NewPagingDTO maps to API paging DTO.
func NewPagingDTO(paginator *paginator.Paginator) PagingDTO {
	dto := PagingDTO{
		TotalItems:  paginator.Nums(),
		TotalPages:  paginator.PageNums(),
		CurrentPage: paginator.Page(),
	}
	if page, err := paginator.PrevPage(); err == nil {
		dto.PreviousPage = &page
	}
	if page, err := paginator.NextPage(); err == nil {
		dto.NextPage = &page
	}
	return dto
}

// PagingDTO holds paging information.
// swagger:model PagingDTO
type PagingDTO struct {
	// The total results.
	TotalItems int `json:"total_items"`
	// The last page of the results.
	TotalPages int `json:"total_pages"`
	// The previous page of the results.
	CurrentPage int `json:"current_page"`
	// The next page of the results.
	PreviousPage *int `json:"previous_page"`
	// The current page of the results.
	NextPage *int `json:"next_page"`
}
