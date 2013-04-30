// Copyright 2013 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package quota implements per-user quota management.
//
// It has a Usage type, that is used to manage generic quotas, and functions
// and methods to interact with the Usage type.
package quota

import (
	"errors"
	"github.com/globocom/tsuru/db"
	"labix.org/v2/mgo"
)

var ErrQuotaAlreadyExists = errors.New("Quota already exists")

// Usage represents the usage of a user. It contains information about the
// limit of items, and the current amount of items in use by the user.
type usage struct {
	// A unique identifier for the user (e.g.: the email).
	User string

	// The slice of items, each identified by a string.
	Items []string

	// The maximum length of Items.
	Limit uint
}

// Create stores a new quota in the database.
func Create(user string, quota uint) error {
	conn, err := db.Conn()
	if err != nil {
		return err
	}
	defer conn.Close()
	err = conn.Quota().Insert(usage{User: user, Limit: quota})
	if e, ok := err.(*mgo.LastError); ok && e.Code == 11000 {
		return ErrQuotaAlreadyExists
	}
	return err
}