/*
 * Alexandria CMDB - Open source configuration management database
 * Copyright (C) 2014  Ryan Armstrong <ryan@cavaliercoder.com>
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
package main

import (
	"testing"
)

func TestFormatFactory(t *testing.T) {
	if format := GetAttributeFormat("i_dont_exist"); format != nil {
		t.Errorf("Expected nil when requesting nonexistant Attribute Format but got: %#v", format)
	}
}

func TestStringFormat(t *testing.T) {
	var err error

	format := GetAttributeFormat("string")
	if format == nil {
		t.Errorf("String attribute format does not appear to be registered")
		return
	}

	att := &CITypeAttribute{
		Name: "Test",
		Type: "notstring",
	}

	var val interface{}

	// Test invalid attribute type
	val = "ShouldPass"
	err = format.Validate(att, &val)
	if err == nil {
		t.Errorf("Expected invalid attribute type to fail but it did not")
	}
	att.Type = "string"

	// Test filters
	att.Filters = []string{"^[a-zA-Z]+$", "^ShouldPass$"}
	err = format.Validate(att, &val)
	if err != nil {
		t.Errorf("Expected string to validate but it did not:\n%s", err.Error())
	}

	val = "ShouldNotPass"
	err = format.Validate(att, &val)
	if err == nil {
		t.Errorf("Expected string to fail validation but it passed")
	}
	att.Filters = nil

	// Test required value
	att.Required = true
	val = ""
	err = format.Validate(att, &val)
	if err == nil {
		t.Errorf("Expected string to fail with a required value but it passed")
	}
	att.Required = false

	// Test minimum length
	att.MinLength = 10
	val = "too short"
	err = format.Validate(att, &val)
	if err == nil {
		t.Errorf("Expected string to fail minimum length requirement but it passed")
	}
	att.MinLength = 0

	// Test maximum length
	att.MaxLength = 7
	val = "too long"
	err = format.Validate(att, &val)
	if err == nil {
		t.Errorf("Expected string to fail maximum length requirement but it passed")
	}
	att.MaxLength = 0

	// Test multiple
	att.MinLength = 17
	att.MaxLength = 17
	att.Required = true
	att.Filters = []string{"^Lorem Ipsum D0lor$", "^[LoremIpsuD0l ]+$", "^[a-zA-Z0-9 ]+$"}
	val = "Lorem Ipsum D0lor"
	err = format.Validate(att, &val)
	if err != nil {
		t.Errorf("Expected string to pass multiple requirements but it failed with:\n    %s", err.Error())
	}
}

func TestNumberFormat(t *testing.T) {
	format := GetAttributeFormat("number")
	if format == nil {
		t.Errorf("Number attribute format does not appear to be registered")
		return
	}

	att := &CITypeAttribute{
		Name: "Test",
		Type: "number",
	}

	vals := []interface{}{
		float64(-12345),
		float64(12345),
		float64(12345.12345),
		float64(-12345.12345),
		"-12345.12345",
		"1.2345e67",
	}

	for _, val := range vals {
		if err := format.Validate(att, &val); err != nil {
			t.Errorf("Expected number %v to validate but it did not:\n%s", val, err.Error())
		}
	}

	// Ensure invalid numbers fail
	var val interface{} = "Not a valid number"
	if err := format.Validate(att, &val); err == nil {
		t.Errorf("Expected non-number '%v' to fail validation but it passed", val)
	}
}

func TestGroupFormat(t *testing.T) {
	format := GetAttributeFormat("group")
	if format == nil {
		t.Errorf("String attribute format does not appear to be registered")
		return
	}

	att := &CITypeAttribute{
		Name:    "Test",
		Type:    "group",
		Filters: []string{"^[a-zA-Z]+$"},
	}

	var err error
	var val interface{}

	val = map[string]interface{}{}
	err = format.Validate(att, &val)
	if err != nil {
		t.Errorf("Expected group to validate but it did not:\n%s", err.Error())
	}

	att.Type = "notgroup"
	err = format.Validate(att, &val)
	if err == nil {
		t.Errorf("Expected invalid attribute type to fail but it did not")
	}
}

func TestTimestampFormat(t *testing.T) {
	format := GetAttributeFormat("timestamp")
	if format == nil {
		t.Errorf("Timestamp attribute format does not appear to be registered")
		return
	}

	var err error
	att := &CITypeAttribute{
		Name: "Test",
		Type: "timestamp",
	}

	// Test some valid dates
	kt := int64(-849135477000)
	tests := []interface{}{
		"-849135477000",                 // Milliseconds since 1970
		"1943-02-04T01:02:03.000Z",      // RFC3339
		"Thu, 04 Feb 1943 01:02:03 GMT", // RFC1123
		"1943-02-04T01:02:03Z",          // ISO 8601
	}

	for _, test := range tests {
		if err = format.Validate(att, &test); err != nil {
			t.Errorf("Expected timestamp '%v' to validate but it failed with: %v", test, err)
		} else {
			// Ensure the validate and updated value matches the desired input
			areEqual(t, test, kt)
		}
	}

	// test invalidate date
	var bad interface{} = "Bad date"
	err = format.Validate(att, &bad)
	if err == nil {
		t.Errorf("Expected timestamp '%v' to fail validation but it passed", bad)
	}
}

func TestBooleanFormat(t *testing.T) {
	format := GetAttributeFormat("boolean")
	if format == nil {
		t.Errorf("Boolean attribute format does not appear to be registered")
		return
	}

	var err error
	att := &CITypeAttribute{
		Name: "Test",
		Type: "boolean",
	}

	var val interface{}

	// Test native booleans
	val = true
	err = format.Validate(att, &val)
	if err != nil {
		t.Errorf("Expected boolean attribute to validate but it did not")
	}

	val = false
	err = format.Validate(att, &val)
	if err != nil {
		t.Errorf("Expected boolean attribute to validate but it did not")
	}

	// Test strings
	val = "TRUE"
	err = format.Validate(att, &val)
	if err != nil {
		t.Error("Expected boolean string to validate but it did not")
	}
	if val != true {
		t.Errorf("Expected native true; got: %#v", val)
	}

	val = "False"
	err = format.Validate(att, &val)
	if err != nil {
		t.Error("Expected boolean string to validate but it did not")
	}
	if val != false {
		t.Errorf("Expected native true; got: %#v", val)
	}

	val = "Not a truthy or falsy"
	err = format.Validate(att, &val)
	if err == nil {
		t.Error("Expected boolean string to validate but it did not")
	}

	// Test numbers
	val = 10
	err = format.Validate(att, &val)
	if err != nil {
		t.Error("Expected boolean number to validate but it did not")
	}
	if val != true {
		t.Errorf("Expected native true; got: %#v", val)
	}

	val = -10
	err = format.Validate(att, &val)
	if err != nil {
		t.Error("Expected boolean number to validate but it did not")
	}
	if val != false {
		t.Errorf("Expected native true; got: %#v", val)
	}

}
