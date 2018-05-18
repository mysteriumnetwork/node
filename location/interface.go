/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package location

// Resolver allows resolving location by ip
type Resolver interface {
	ResolveCountry(ip string) (string, error)
}

// Detector allows detecting location by current ip
type Detector interface {
	DetectLocation() (Location, error)
}

// Cache allows caching location
type Cache interface {
	Get() Location
	RefreshAndGet() (Location, error)
}
