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
	"github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha4"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (in *MaasCluster) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.MaasCluster)

	return Convert_v1alpha3_MaasCluster_To_v1alpha4_MaasCluster(in, dst, nil)
}

func (in *MaasCluster) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.MaasCluster)

	return Convert_v1alpha4_MaasCluster_To_v1alpha3_MaasCluster(src, in, nil)
}

func (in *MaasClusterList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.MaasClusterList)

	return Convert_v1alpha3_MaasClusterList_To_v1alpha4_MaasClusterList(in, dst, nil)
}

func (in *MaasClusterList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.MaasClusterList)

	return Convert_v1alpha4_MaasClusterList_To_v1alpha3_MaasClusterList(src, in, nil)
}

func (in *MaasMachine) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.MaasMachine)

	return Convert_v1alpha3_MaasMachine_To_v1alpha4_MaasMachine(in, dst, nil)
}

func (in *MaasMachine) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.MaasMachine)

	return Convert_v1alpha4_MaasMachine_To_v1alpha3_MaasMachine(src, in, nil)
}

func (in *MaasMachineList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.MaasMachineList)

	return Convert_v1alpha3_MaasMachineList_To_v1alpha4_MaasMachineList(in, dst, nil)
}

func (in *MaasMachineList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.MaasMachineList)

	return Convert_v1alpha4_MaasMachineList_To_v1alpha3_MaasMachineList(src, in, nil)
}

func (in *MaasMachineTemplate) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.MaasMachineTemplate)

	return Convert_v1alpha3_MaasMachineTemplate_To_v1alpha4_MaasMachineTemplate(in, dst, nil)
}

func (in *MaasMachineTemplate) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.MaasMachineTemplate)

	return Convert_v1alpha4_MaasMachineTemplate_To_v1alpha3_MaasMachineTemplate(src, in, nil)
}

func (in *MaasMachineTemplateList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha4.MaasMachineTemplateList)

	return Convert_v1alpha3_MaasMachineTemplateList_To_v1alpha4_MaasMachineTemplateList(in, dst, nil)
}

func (in *MaasMachineTemplateList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha4.MaasMachineTemplateList)

	return Convert_v1alpha4_MaasMachineTemplateList_To_v1alpha3_MaasMachineTemplateList(src, in, nil)
}
