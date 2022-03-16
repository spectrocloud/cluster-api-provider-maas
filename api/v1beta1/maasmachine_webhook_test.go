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

package v1beta1

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMaasMachine_ValidateUpdate(t *testing.T) {
	cpuBefore := 10
	cpuAfter := 11
	memoryBefore := 100
	memoryAfter := 101

	tests := []struct {
		name       string
		oldMachine *MaasMachine
		newMachine *MaasMachine
		wantErr    bool
	}{
		{
			name: "change in min memory, cpu or image should not be allowed",
			oldMachine: &MaasMachine{
				Spec: MaasMachineSpec{
					MinCPU:        &cpuBefore,
					MinMemoryInMB: &memoryBefore,
					Image:         "ubuntu1804-k8s-1.19",
				},
			},
			newMachine: &MaasMachine{
				Spec: MaasMachineSpec{
					MinCPU:        &cpuBefore,
					MinMemoryInMB: &memoryAfter,
					Image:         "ubuntu1804-k8s-1.19",
				},
			},
			wantErr: true,
		},
		{
			name: "change in min memory, cpu or image should not be allowed",
			oldMachine: &MaasMachine{
				Spec: MaasMachineSpec{
					MinCPU:        &cpuBefore,
					MinMemoryInMB: &memoryBefore,
					Image:         "ubuntu1804-k8s-1.19",
				},
			},
			newMachine: &MaasMachine{
				Spec: MaasMachineSpec{
					MinCPU:        &cpuAfter,
					MinMemoryInMB: &memoryBefore,
					Image:         "ubuntu1804-k8s-1.19",
				},
			},
			wantErr: true,
		},
		{
			name: "change in min memory, cpu or image should not be allowed",
			oldMachine: &MaasMachine{
				Spec: MaasMachineSpec{
					MinCPU:        &cpuBefore,
					MinMemoryInMB: &memoryBefore,
					Image:         "ubuntu1804-k8s-1.19",
				},
			},
			newMachine: &MaasMachine{
				Spec: MaasMachineSpec{
					MinCPU:        &cpuBefore,
					MinMemoryInMB: &memoryBefore,
					Image:         "ubuntu1804-k8s-1.20",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		ctx := context.TODO()
		t.Run(tt.name, func(t *testing.T) {
			machine := tt.oldMachine.DeepCopy()
			machine.ObjectMeta = metav1.ObjectMeta{
				GenerateName: "machine-",
				Namespace:    "default",
			}
			if err := testEnv.Create(ctx, machine); err != nil {
				t.Errorf("failed to create machine: %v", err)
			}
			machine.Spec = tt.newMachine.Spec
			if err := testEnv.Update(ctx, machine); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
