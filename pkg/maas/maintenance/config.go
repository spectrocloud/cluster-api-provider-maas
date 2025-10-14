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
package maintenance

import "time"

type ForcePolicy string

const (
	ForceHalt  ForcePolicy = "halt"
	ForceAllow ForcePolicy = "force"
	ForceRelax ForcePolicy = "relax"
)

type HMCConfig struct {
	PerWLCMoveTimeout  time.Duration
	PerHostWaveTimeout time.Duration
	Policy             ForcePolicy
}

type VECConfig struct {
	MoveTimeout  time.Duration
	RetryBackoff time.Duration
	CheckAPIPath string
}

func DefaultHMCConfig() HMCConfig {
	return HMCConfig{
		PerWLCMoveTimeout:  20 * time.Minute,
		PerHostWaveTimeout: 60 * time.Minute,
		Policy:             ForceHalt,
	}
}

func DefaultVECConfig() VECConfig {
	return VECConfig{
		MoveTimeout:  20 * time.Minute,
		RetryBackoff: 30 * time.Second,
		CheckAPIPath: "/readyz",
	}
}
