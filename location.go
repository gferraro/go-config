// go-config - Library for reading cacophony config files.
// Copyright (C) 2018, The Cacophony Project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package config

const LocationKey = "location"

type Location struct {
	Accuracy  int
	Altitude  int
	Latitude  float32
	Longitude float32
}

// Default location used when setting windows relative to sunset/sunrise
func DefaultWindowLocation() Location {
	return Location{
		Latitude:  -43.5321,
		Longitude: 172.6362,
	}
}
