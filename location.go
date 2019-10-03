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

import (
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
)

func init() {
	allSections[LocationKey] = section{
		key:         LocationKey,
		structToMap: locationToMap,
		mapToStruct: mapToLocation,
		validate:    validateLocation,
	}
}

const LocationKey = "location"

type Location struct {
	Timestamp time.Time
	Accuracy  float32
	Altitude  float32
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

func locationToMap(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != mapStrInterfaceType {
		return data, nil
	}
	switch f {
	case reflect.TypeOf(&Location{}):
		data = *(data.(*Location)) // follow the pointer
		fallthrough
	case reflect.TypeOf(Location{}):
		m := map[string]interface{}{}
		err := mapstructure.Decode(data, &m)
		m["Timestamp"] = data.(Location).Timestamp.Truncate(time.Second)
		return m, err
	default:
		return data, nil
	}
}

func stringToTime(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != reflect.TypeOf(time.Time{}) {
		return data, nil
	}
	switch f {
	case reflect.TypeOf(""):
		layout := "2006-01-02 15:04:05.000000000 -0700 MST"
		return time.Parse(layout, data.(string))
	}
	return data, nil
}

func mapToLocation(m map[string]interface{}) (interface{}, error) {
	var l Location
	decoderConfig := mapstructure.DecoderConfig{
		DecodeHook:       stringToTime,
		Result:           &l,
		WeaklyTypedInput: true,
	}
	decoder, err := mapstructure.NewDecoder(&decoderConfig)
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(m); err != nil {
		return nil, err
	}
	return l, nil
}

func validateLocation(l interface{}) bool {
	return true
}
