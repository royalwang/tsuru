// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"fmt"
	"github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/auth"
	"github.com/tsuru/tsuru/errors"
	"github.com/tsuru/tsuru/io"
	"github.com/tsuru/tsuru/log"
	"net/http"
)

const (
	tsuruMin      = "0.9.0"
	craneMin      = "0.5.1"
	tsuruAdminMin = "0.3.0"
)

func setVersionHeaders(w http.ResponseWriter) {
	w.Header().Set("Supported-Tsuru", tsuruMin)
	w.Header().Set("Supported-Crane", craneMin)
	w.Header().Set("Supported-Tsuru-Admin", tsuruAdminMin)
}

type Handler func(http.ResponseWriter, *http.Request) error

func (fn Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setVersionHeaders(w)
	defer func() {
		if r.Body != nil {
			r.Body.Close()
		}
	}()
	fw := io.FlushingWriter{ResponseWriter: w}
	if err := fn(&fw, r); err != nil {
		if fw.Wrote() {
			fmt.Fprintln(&fw, err)
		} else {
			http.Error(&fw, err.Error(), http.StatusInternalServerError)
		}
		log.Error(err.Error())
	}
}

func validate(token string, r *http.Request) (auth.Token, error) {
	if token == "" {
		return nil, &errors.HTTP{
			Message: "You must provide the Authorization header",
		}
	}
	invalid := &errors.HTTP{Message: "Invalid token"}
	t, err := app.AuthScheme.Auth(token)
	if err != nil {
		return nil, invalid
	}
	if t.IsAppToken() {
		if q := r.URL.Query().Get(":app"); q != "" && t.GetAppName() != q {
			return nil, invalid
		}
	}
	return t, nil
}

type authorizationRequiredHandler func(http.ResponseWriter, *http.Request, auth.Token) error

func (fn authorizationRequiredHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setVersionHeaders(w)
	defer func() {
		if r.Body != nil {
			r.Body.Close()
		}
	}()
	fw := io.FlushingWriter{ResponseWriter: w}
	token := r.Header.Get("Authorization")
	if t, err := validate(token, r); err != nil {
		http.Error(&fw, err.Error(), http.StatusUnauthorized)
	} else if err = fn(&fw, r, t); err != nil {
		code := http.StatusInternalServerError
		if e, ok := err.(*errors.HTTP); ok {
			code = e.Code
		}
		if fw.Wrote() {
			fmt.Fprintln(&fw, err)
		} else {
			http.Error(&fw, err.Error(), code)
		}
		log.Error(err.Error())
	}
}

type AdminRequiredHandler authorizationRequiredHandler

func (fn AdminRequiredHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setVersionHeaders(w)
	defer func() {
		if r.Body != nil {
			r.Body.Close()
		}
	}()
	fw := io.FlushingWriter{ResponseWriter: w}
	header := r.Header.Get("Authorization")
	if header == "" {
		http.Error(&fw, "You must provide the Authorization header", http.StatusUnauthorized)
	} else if t, err := app.AuthScheme.Auth(header); err != nil {
		http.Error(&fw, "Invalid token", http.StatusUnauthorized)
	} else if user, err := t.User(); err != nil || !user.IsAdmin() {
		http.Error(&fw, "Forbidden", http.StatusForbidden)
	} else if err = fn(&fw, r, t); err != nil {
		code := http.StatusInternalServerError
		if e, ok := err.(*errors.HTTP); ok {
			code = e.Code
		}
		if fw.Wrote() {
			fmt.Fprintln(&fw, err)
		} else {
			http.Error(&fw, err.Error(), code)
		}
		log.Error(err.Error())
	}
}
