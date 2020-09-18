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
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

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
