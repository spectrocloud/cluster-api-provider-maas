/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"testing"
)

func TestStableHashStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input1   []string
		input2   []string
		wantSame bool
	}{
		{
			name:     "identical lists same order",
			input1:   []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"},
			input2:   []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"},
			wantSame: true,
		},
		{
			name:     "identical lists different order",
			input1:   []string{"3.3.3.3", "1.1.1.1", "2.2.2.2"},
			input2:   []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"},
			wantSame: true,
		},
		{
			name:     "different lists",
			input1:   []string{"1.1.1.1", "2.2.2.2"},
			input2:   []string{"1.1.1.1", "3.3.3.3"},
			wantSame: false,
		},
		{
			name:     "empty vs populated",
			input1:   []string{},
			input2:   []string{"1.1.1.1"},
			wantSame: false,
		},
		{
			name:     "both empty",
			input1:   []string{},
			input2:   []string{},
			wantSame: true,
		},
		{
			name:     "with empty strings filtered",
			input1:   []string{"1.1.1.1", "", "2.2.2.2"},
			input2:   []string{"2.2.2.2", "1.1.1.1"},
			wantSame: true,
		},
		{
			name:     "nil vs empty",
			input1:   nil,
			input2:   []string{},
			wantSame: true,
		},
		{
			name:     "duplicates in input",
			input1:   []string{"1.1.1.1", "1.1.1.1", "2.2.2.2"},
			input2:   []string{"1.1.1.1", "2.2.2.2", "1.1.1.1"},
			wantSame: true,
		},
		{
			name:     "single element",
			input1:   []string{"1.1.1.1"},
			input2:   []string{"1.1.1.1"},
			wantSame: true,
		},
		{
			name:     "different single element",
			input1:   []string{"1.1.1.1"},
			input2:   []string{"2.2.2.2"},
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := StableHashStringSlice(tt.input1)
			hash2 := StableHashStringSlice(tt.input2)

			if tt.wantSame {
				if hash1 != hash2 {
					t.Errorf("StableHashStringSlice() hashes should be equal\ngot hash1=%s\ngot hash2=%s", hash1, hash2)
				}
			} else {
				if hash1 == hash2 {
					t.Errorf("StableHashStringSlice() hashes should be different\ngot hash1=%s\ngot hash2=%s", hash1, hash2)
				}
			}

			// Verify hash is stable on repeated calls
			hash1Again := StableHashStringSlice(tt.input1)
			if hash1 != hash1Again {
				t.Errorf("StableHashStringSlice() not stable on repeated calls\nfirst=%s\nsecond=%s", hash1, hash1Again)
			}
		})
	}
}

func TestStableHashStringSlice_Properties(t *testing.T) {
	t.Run("hash is non-empty", func(t *testing.T) {
		hash := StableHashStringSlice([]string{"1.1.1.1"})
		if hash == "" {
			t.Error("StableHashStringSlice() returned empty hash")
		}
	})

	t.Run("hash is hexadecimal", func(t *testing.T) {
		hash := StableHashStringSlice([]string{"1.1.1.1"})
		// SHA256 hex should be 64 characters
		if len(hash) != 64 {
			t.Errorf("StableHashStringSlice() hash length = %d, want 64 (SHA256 hex)", len(hash))
		}
		// Check it's valid hex
		for _, c := range hash {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("StableHashStringSlice() hash contains non-hex character: %c", c)
				break
			}
		}
	})
}
