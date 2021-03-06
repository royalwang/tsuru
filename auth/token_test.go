// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"crypto"
	"labix.org/v2/mgo/bson"
	"launchpad.net/gocheck"
	"sync"
	"time"
)

func (s *S) TestTokenCannotRepeat(c *gocheck.C) {
	input := "user-token"
	tokens := make([]string, 10)
	var wg sync.WaitGroup
	for i := range tokens {
		wg.Add(1)
		go func(i int) {
			tokens[i] = token(input, crypto.MD5)
			wg.Done()
		}(i)
	}
	wg.Wait()
	reference := tokens[0]
	for _, t := range tokens[1:] {
		c.Check(t, gocheck.Not(gocheck.Equals), reference)
	}
}

func (s *S) TestCreatePasswordToken(c *gocheck.C) {
	u := User{Email: "pure@alanis.com"}
	t, err := createPasswordToken(&u)
	c.Assert(err, gocheck.IsNil)
	c.Assert(t.UserEmail, gocheck.Equals, u.Email)
	c.Assert(t.Used, gocheck.Equals, false)
	var dbToken passwordToken
	err = s.conn.PasswordTokens().Find(bson.M{"_id": t.Token}).One(&dbToken)
	c.Assert(err, gocheck.IsNil)
	c.Assert(dbToken.Token, gocheck.Equals, t.Token)
	c.Assert(dbToken.UserEmail, gocheck.Equals, t.UserEmail)
	c.Assert(dbToken.Used, gocheck.Equals, t.Used)
}

func (s *S) TestCreatePasswordTokenErrors(c *gocheck.C) {
	var tests = []struct {
		input *User
		want  string
	}{
		{nil, "User is nil"},
		{&User{}, "User email is empty"},
	}
	for _, t := range tests {
		token, err := createPasswordToken(t.input)
		c.Check(token, gocheck.IsNil)
		c.Check(err, gocheck.NotNil)
		c.Check(err.Error(), gocheck.Equals, t.want)
	}
}

func (s *S) TestPasswordTokenUser(c *gocheck.C) {
	u := User{Email: "need@who.com", Password: "123456"}
	err := u.Create()
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Users().Remove(bson.M{"email": u.Email})
	t, err := createPasswordToken(&u)
	c.Assert(err, gocheck.IsNil)
	u2, err := t.user()
	u2.Keys = u.Keys
	c.Assert(err, gocheck.IsNil)
	c.Assert(*u2, gocheck.DeepEquals, u)
}

func (s *S) TestGetPasswordToken(c *gocheck.C) {
	u := User{Email: "porcelain@opeth.com"}
	t, err := createPasswordToken(&u)
	c.Assert(err, gocheck.IsNil)
	t2, err := getPasswordToken(t.Token)
	t2.Creation = t.Creation
	c.Assert(err, gocheck.IsNil)
	c.Assert(t2, gocheck.DeepEquals, t)
}

func (s *S) TestGetPasswordTokenUnknown(c *gocheck.C) {
	t, err := getPasswordToken("what??")
	c.Assert(t, gocheck.IsNil)
	c.Assert(err, gocheck.Equals, ErrInvalidToken)
}

func (s *S) TestGetPasswordUsedToken(c *gocheck.C) {
	u := User{Email: "porcelain@opeth.com"}
	t, err := createPasswordToken(&u)
	c.Assert(err, gocheck.IsNil)
	t.Used = true
	err = s.conn.PasswordTokens().UpdateId(t.Token, t)
	c.Assert(err, gocheck.IsNil)
	t2, err := getPasswordToken(t.Token)
	c.Assert(t2, gocheck.IsNil)
	c.Assert(err, gocheck.Equals, ErrInvalidToken)
}

func (s *S) TestPasswordTokensAreValidFor24Hours(c *gocheck.C) {
	u := User{Email: "porcelain@opeth.com"}
	t, err := createPasswordToken(&u)
	c.Assert(err, gocheck.IsNil)
	t.Creation = time.Now().Add(-24 * time.Hour)
	err = s.conn.PasswordTokens().UpdateId(t.Token, t)
	c.Assert(err, gocheck.IsNil)
	t2, err := getPasswordToken(t.Token)
	c.Assert(t2, gocheck.IsNil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "Invalid token")
}
