/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rbac

import "github.com/go-chassis/cari/pkg/errsvc"

// BatchCreateAccountsRequest the request definition of batch create accounts
type BatchCreateAccountsRequest struct {
	Accounts []*Account `json:"accounts"`
}

// BatchCreateAccountsResponse the response definition of batch create accounts
type BatchCreateAccountsResponse struct {
	Accounts []*BatchCreateAccountItemResponse `json:"accounts"`
}

// BatchCreateAccountItemResponse the item result of batch create accounts
type BatchCreateAccountItemResponse struct {
	Name string `json:"name"`

	*errsvc.Error
}

type AccountResponse struct {
	Total    int64      `json:"total,omitempty"`
	Accounts []*Account `json:"data,omitempty"`
}

type Account struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
	//Deprecated
	Role                string   `json:"role,omitempty"`
	Roles               []string `json:"roles,omitempty"`
	TokenExpirationTime string   `json:"tokenExpirationTime,omitempty" bson:"token_expiration_time"`
	CurrentPassword     string   `json:"currentPassword,omitempty" bson:"current_password"`
	Status              string   `json:"status,omitempty"`
	CreateTime          string   `json:"createTime,omitempty"`
	UpdateTime          string   `json:"updateTime,omitempty"`
}

func (a *Account) Check() error {
	if a.Name == a.Password {
		return ErrSameAsName
	}
	if reverse(a.Name) == a.Password {
		return ErrSameAsReversedName
	}
	return nil
}
func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

type Token struct {
	TokenStr string `json:"token,omitempty"`
}

type AuthUser struct {
	Username string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
}
