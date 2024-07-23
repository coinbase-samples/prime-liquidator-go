/**
 * Copyright 2024-present Coinbase Global, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package monitor

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestMeetsTwapRequirements(t *testing.T) {

	cases := []struct {
		description            string
		value                  decimal.Decimal
		twapMinNotionalPerHour int
		twapDuration           time.Duration
		expected               bool
	}{
		{
			description:            "TestMeetsTwapRequirements0",
			value:                  decimal.NewFromFloat(100),
			twapMinNotionalPerHour: 100,
			twapDuration:           60 * time.Minute,
			expected:               true,
		},
		{
			description:            "TestMeetsTwapRequirements0",
			value:                  decimal.NewFromFloat(99),
			twapMinNotionalPerHour: 100,
			twapDuration:           60 * time.Minute,
			expected:               false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			result := meetsTwapRequirements(tt.value, tt.twapMinNotionalPerHour, tt.twapDuration)
			if result != tt.expected {
				t.Errorf("test: %s - expected: %t - received: %t", tt.description, tt.expected, result)
			}
		})
	}
}
