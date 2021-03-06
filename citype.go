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
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	ciTypeCollection = "citypes"
)

type CIType struct {
	model `json:"-" bson:",inline"`

	Name        string              `json:"name,omitempty"`
	ShortName   string              `json:"shortName,omitempty"`
	Description string              `json:"description,omitempty" xml:",omitempty" bson:",omitempty"`
	Attributes  CITypeAttributeList `json:"attributes,omitempty" xml:"attribute"`
}

type CITypeAttribute struct {
	Name        string              `json:"name" xml:",attr"`
	ShortName   string              `json:"shortName" xml:",attr"`
	Type        string              `json:"type" xml:",attr"`
	Description string              `json:"description,omitempty" xml:",omitempty" bson:",omitempty"`
	Children    CITypeAttributeList `json:"children,omitempty" xml:"attribute,omitempty" bson:",omitempty"`

	// Common Options
	Required bool `json:"required,omitempty" xml:",omitempty" bson:",omitempty"`

	IsArray  bool `json:"isArray,omitempty" xml:",omitempty" bson:",omitempty"`
	MinCount int  `json:"minCount,omitempty" xml:",omitempty" bson:",omitempty"`
	MaxCount int  `json:"maxCount,omitempty" xml:",omitempty" bson:",omitempty"`

	// Group options
	Singular string `json:"singular,omitempty" xml:",omitempty" bson:",omitempty"`

	// String options
	MinLength int      `json:"minLength,omitempty" xml:",omitempty" bson:",omitempty"`
	MaxLength int      `json:"maxLength,omitempty" xml:",omitempty" bson:",omitempty"`
	Filters   []string `json:"filters,omitempty" xml:",omitempty" bson:",omitempty"`

	// Number options
	Units    string  `json:"units,omitempty" xml:",omitempty" bson:",omitempty"`
	MinValue float64 `json:"minValue,omitempty" xml:",omitempty" bson:",omitempty"`
	MaxValue float64 `json:"maxValue,omitempty" xml:",omitempty" bson:",omitempty"`
}

type CITypeAttributeList []CITypeAttribute

func (c *CITypeAttributeList) Get(name string) *CITypeAttribute {
	name = strings.ToLower(name)
	for _, att := range *c {
		if att.ShortName == name {
			return &att
		}
	}

	return nil
}

func (c *CIType) Validate() error {
	if c.Name == "" {
		return errors.New("No CI Type name specified")
	}

	if c.ShortName == "" {
		c.ShortName = GetShortName(c.Name)
	}

	if !IsValidShortName(c.ShortName) {
		return errors.New("Invalid characters in CI Type name")
	}

	// Validate each attribute
	err := c.validateAttributes(&c.Attributes, "")
	if err != nil {
		return err
	}

	return nil
}

func (c *CIType) validateAttributes(atts *CITypeAttributeList, path string) error {
	for index, _ := range *atts {
		// Derefence the attribute so it may be modified
		att := &(*atts)[index]

		if att.Name == "" {
			return errors.New("No attribute name specified")
		}

		att.ShortName = GetShortName(att.Name)
		if !IsValidShortName(att.ShortName) {
			return errors.New(fmt.Sprintf("Invalid characters in CI Attribute '%s%s'", path, att.ShortName))
		}

		// Validate format
		if att.Type == "" {
			return errors.New(fmt.Sprintf("No type specified for CI Attribute '%s%s'", path, att.ShortName))
		}

		if GetAttributeFormat(att.Type) == nil {
			return errors.New(fmt.Sprintf("Unsupported attribute format '%s' for CI Attribute '%s%s'", att.Type, path, att.ShortName))
		}

		// Validate children
		if att.Type == "group" {
			err := c.validateAttributes(&att.Children, fmt.Sprintf("%s.", att.ShortName))
			if err != nil {
				return err
			}
		} else if len(att.Children) > 0 {
			return errors.New(fmt.Sprintf("CI Attribute '%s%s' has children but is not a group attribute", path, att.ShortName))
		}
	}

	return nil
}

func GetCITypes(res http.ResponseWriter, req *http.Request) {
	// Get CMDB details
	cmdb := GetPathVar(req, "cmdb")
	db := GetCmdbBackend(req, cmdb)
	if db == nil {
		ErrNotFound(res, req)
		return
	}

	// Build query
	sel, err := GetRequestSelecter(req)
	if err != nil {
		ErrBadRequest(res, req, err)
	}
	query := db.C(ciTypeCollection).Find(nil)
	if sel != nil {
		query = query.Select(sel)
	}

	var citypes []CIType
	err = query.All(&citypes)
	if Handle(res, req, err) {
		return
	}

	Render(res, req, http.StatusOK, citypes)
}

func GetCITypeByName(res http.ResponseWriter, req *http.Request) {
	// Get CMDB details
	cmdb := strings.ToLower(GetPathVar(req, "cmdb"))
	db := GetCmdbBackend(req, cmdb)
	if db == nil {
		ErrNotFound(res, req)
		return
	}

	// Build query
	name := GetPathVar(req, "name")
	sel, err := GetRequestSelecter(req)
	if err != nil {
		ErrBadRequest(res, req, err)
	}

	query := db.C(ciTypeCollection).Find(M{"shortname": name})
	if sel != nil {
		query = query.Select(sel)
	}

	// Get the type
	var citype CIType
	err = query.One(&citype)
	if Handle(res, req, err) {
		return
	}

	Render(res, req, http.StatusOK, citype)
}

func AddCIType(res http.ResponseWriter, req *http.Request) {
	// Parse request into CIType
	var citype CIType
	err := Bind(req, &citype)
	if Handle(res, req, err) {
		return
	}
	citype.InitModel()

	// Validate
	err = citype.Validate()
	if err != nil {
		ErrBadRequest(res, req, err)
		return
	}

	// Get CMDB details
	cmdb := GetPathVar(req, "cmdb")
	db := GetCmdbBackend(req, cmdb)
	if db == nil {
		ErrNotFound(res, req)
		return
	}

	// Insert new type
	err = db.C(ciTypeCollection).Insert(&citype)
	if Handle(res, req, err) {
		return
	}

	RenderCreated(res, req, V1Uri(fmt.Sprintf("/cmdbs/%s/citypes/%s", cmdb, citype.ShortName)))
}

func UpdateCITypeByName(res http.ResponseWriter, req *http.Request) {
	// Parse request into CIType
	var citype CIType
	err := Bind(req, &citype)
	if Handle(res, req, err) {
		return
	}

	// Skip InitModel() but still validate
	err = citype.Validate()
	if err != nil {
		ErrBadRequest(res, req, err)
		return
	}

	// Get CMDB details
	cmdb := GetPathVar(req, "cmdb")
	db := GetCmdbBackend(req, cmdb)
	if db == nil {
		ErrNotFound(res, req)
		return
	}

	// Fetch the original CIType
	name := GetPathVar(req, "name")
	var orig CIType
	err = db.C(ciTypeCollection).Find(M{"shortname": name}).One(&orig)
	if Handle(res, req, err) {
		return
	}

	// Prepare the new record
	citype.Id = orig.Id
	citype.Created = orig.Created
	citype.ShortName = GetShortName(citype.Name)
	citype.InitModel()

	// Update
	err = db.C(ciTypeCollection).Update(M{"_id": orig.Id}, &citype)
	if Handle(res, req, err) {
		return
	}

	// Compute the new URL
	location := ""
	if citype.ShortName != orig.ShortName {
		location = V1Uri(fmt.Sprintf("/cmdbs/%s/citypes/%s", cmdb, citype.ShortName))
	}

	RenderUpdated(res, req, location)
}

func DeleteCITypeByName(res http.ResponseWriter, req *http.Request) {
	cmdb := GetPathVar(req, "cmdb")
	name := GetPathVar(req, "name")

	// Get CMDB details
	db := GetCmdbBackend(req, cmdb)
	if db == nil {
		ErrNotFound(res, req)
		return
	}

	// Remove CI Type entry
	err := db.C(ciTypeCollection).Remove(M{"shortname": name})
	if Handle(res, req, err) {
		return
	}

	// Remove associated CI collection
	err = db.C(name).DropCollection()
	if Handle(res, req, err) {
		return
	}

	Render(res, req, http.StatusNoContent, "")
}
