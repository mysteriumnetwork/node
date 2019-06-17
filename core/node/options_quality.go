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

package node

// QualityType identifies Quality Oracle provider
type QualityType string

const (
	// QualityTypeElastic defines type which uses ElasticSearch as Quality Oracle provider
	QualityTypeElastic = QualityType("elastic")
	// QualityTypeMORQA defines type which uses Mysterium MORQA as Quality Oracle provider
	QualityTypeMORQA = QualityType("morqa")
	// QualityTypeNone defines type which disables Quality Oracle
	QualityTypeNone = QualityType("none")
)

// OptionsQuality describes possible parameters of Quality Oracle configuration
type OptionsQuality struct {
	Type    QualityType
	Address string
}
