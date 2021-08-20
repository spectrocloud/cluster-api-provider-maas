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

package v1alpha3

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"

	"github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha4"
)

func fuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		MAASMachineFuzzer,
		MAASMachineTemplateFuzzer,
	}
}

func MAASMachineFuzzer(obj *MaasMachine, c fuzz.Continue) {
	c.FuzzNoCustom(obj)

	// MAASMachine MinMemory and MinCPU change from optional to mandatory
	defMem := 8192
	obj.Spec.MinMemory = &defMem
	defCPU := 4
	obj.Spec.MinCPU = &defCPU
}

func MAASMachineTemplateFuzzer(obj *MaasMachineTemplate, c fuzz.Continue) {
	c.FuzzNoCustom(obj)

	// MAASMachine MinMemory and MinCPU change from optional to demandatory
	defMem := 8192
	obj.Spec.Template.Spec.MinMemory = &defMem
	defCPU := 4
	obj.Spec.Template.Spec.MinCPU = &defCPU
}

func TestFuzzyConversion(t *testing.T) {
	g := NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(Succeed())
	g.Expect(v1alpha4.AddToScheme(scheme)).To(Succeed())

	t.Run("for MaasCluster", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &v1alpha4.MaasCluster{},
		Spoke:  &MaasCluster{},
	}))

	t.Run("for MaasMachine", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &v1alpha4.MaasMachine{},
		Spoke:       &MaasMachine{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))

	t.Run("for MaasMachineTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &v1alpha4.MaasMachineTemplate{},
		Spoke:       &MaasMachineTemplate{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))
}
