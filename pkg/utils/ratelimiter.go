/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2020 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package utils

import (
	"net/http"
	"sync"

	"go.uber.org/ratelimit"
)

type limitTransport struct {
	http.RoundTripper
	limiter ratelimit.Limiter
}

var _ http.RoundTripper = limitTransport{}

func (t limitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.limiter.Take()
	return t.RoundTripper.RoundTrip(req)
}

var limiterOnce sync.Once
var limiter ratelimit.Limiter

// NewLimiter generates a new limiter.
func NewLimiter(rate int) ratelimit.Limiter {
	limiterOnce.Do(func() {
		limiter = ratelimit.New(rate)
	})
	return limiter
}

// NewRateLimitedTransport generates a new transport with rateLimit.
func NewRateLimitedTransport(rate int, transport http.RoundTripper) http.RoundTripper {
	return &limitTransport{
		RoundTripper: transport,
		limiter:      NewLimiter(rate),
	}
}

var listLimiterOnce sync.Once
var listLimiter ratelimit.Limiter

// NewListLimiter generates a new limiter.
func NewListLimiter(rate int) ratelimit.Limiter {
	listLimiterOnce.Do(func() {
		listLimiter = ratelimit.New(rate)
	})
	return listLimiter
}

// NewListRateLimitedTransport generates a new transport with rateLimit.
func NewListRateLimitedTransport(rate int, transport http.RoundTripper) http.RoundTripper {
	return &limitTransport{
		RoundTripper: transport,
		limiter:      NewListLimiter(rate),
	}
}
