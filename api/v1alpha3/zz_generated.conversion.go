// +build !ignore_autogenerated_core_v1alpha3

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

// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha3

import (
	unsafe "unsafe"

	v1alpha4 "github.com/spectrocloud/cluster-api-provider-maas/api/v1alpha4"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	apiv1alpha4 "sigs.k8s.io/cluster-api/api/v1alpha4"
	errors "sigs.k8s.io/cluster-api/errors"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*APIEndpoint)(nil), (*v1alpha4.APIEndpoint)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_APIEndpoint_To_v1alpha4_APIEndpoint(a.(*APIEndpoint), b.(*v1alpha4.APIEndpoint), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.APIEndpoint)(nil), (*APIEndpoint)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_APIEndpoint_To_v1alpha3_APIEndpoint(a.(*v1alpha4.APIEndpoint), b.(*APIEndpoint), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasCluster)(nil), (*v1alpha4.MaasCluster)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasCluster_To_v1alpha4_MaasCluster(a.(*MaasCluster), b.(*v1alpha4.MaasCluster), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasCluster)(nil), (*MaasCluster)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasCluster_To_v1alpha3_MaasCluster(a.(*v1alpha4.MaasCluster), b.(*MaasCluster), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasClusterList)(nil), (*v1alpha4.MaasClusterList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasClusterList_To_v1alpha4_MaasClusterList(a.(*MaasClusterList), b.(*v1alpha4.MaasClusterList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasClusterList)(nil), (*MaasClusterList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasClusterList_To_v1alpha3_MaasClusterList(a.(*v1alpha4.MaasClusterList), b.(*MaasClusterList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasClusterSpec)(nil), (*v1alpha4.MaasClusterSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasClusterSpec_To_v1alpha4_MaasClusterSpec(a.(*MaasClusterSpec), b.(*v1alpha4.MaasClusterSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasClusterSpec)(nil), (*MaasClusterSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasClusterSpec_To_v1alpha3_MaasClusterSpec(a.(*v1alpha4.MaasClusterSpec), b.(*MaasClusterSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasClusterStatus)(nil), (*v1alpha4.MaasClusterStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasClusterStatus_To_v1alpha4_MaasClusterStatus(a.(*MaasClusterStatus), b.(*v1alpha4.MaasClusterStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasClusterStatus)(nil), (*MaasClusterStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasClusterStatus_To_v1alpha3_MaasClusterStatus(a.(*v1alpha4.MaasClusterStatus), b.(*MaasClusterStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasMachine)(nil), (*v1alpha4.MaasMachine)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasMachine_To_v1alpha4_MaasMachine(a.(*MaasMachine), b.(*v1alpha4.MaasMachine), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasMachine)(nil), (*MaasMachine)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasMachine_To_v1alpha3_MaasMachine(a.(*v1alpha4.MaasMachine), b.(*MaasMachine), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasMachineList)(nil), (*v1alpha4.MaasMachineList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasMachineList_To_v1alpha4_MaasMachineList(a.(*MaasMachineList), b.(*v1alpha4.MaasMachineList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasMachineList)(nil), (*MaasMachineList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasMachineList_To_v1alpha3_MaasMachineList(a.(*v1alpha4.MaasMachineList), b.(*MaasMachineList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasMachineStatus)(nil), (*v1alpha4.MaasMachineStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasMachineStatus_To_v1alpha4_MaasMachineStatus(a.(*MaasMachineStatus), b.(*v1alpha4.MaasMachineStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasMachineStatus)(nil), (*MaasMachineStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasMachineStatus_To_v1alpha3_MaasMachineStatus(a.(*v1alpha4.MaasMachineStatus), b.(*MaasMachineStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasMachineTemplate)(nil), (*v1alpha4.MaasMachineTemplate)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasMachineTemplate_To_v1alpha4_MaasMachineTemplate(a.(*MaasMachineTemplate), b.(*v1alpha4.MaasMachineTemplate), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasMachineTemplate)(nil), (*MaasMachineTemplate)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasMachineTemplate_To_v1alpha3_MaasMachineTemplate(a.(*v1alpha4.MaasMachineTemplate), b.(*MaasMachineTemplate), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasMachineTemplateList)(nil), (*v1alpha4.MaasMachineTemplateList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasMachineTemplateList_To_v1alpha4_MaasMachineTemplateList(a.(*MaasMachineTemplateList), b.(*v1alpha4.MaasMachineTemplateList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasMachineTemplateList)(nil), (*MaasMachineTemplateList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasMachineTemplateList_To_v1alpha3_MaasMachineTemplateList(a.(*v1alpha4.MaasMachineTemplateList), b.(*MaasMachineTemplateList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasMachineTemplateResource)(nil), (*v1alpha4.MaasMachineTemplateResource)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasMachineTemplateResource_To_v1alpha4_MaasMachineTemplateResource(a.(*MaasMachineTemplateResource), b.(*v1alpha4.MaasMachineTemplateResource), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasMachineTemplateResource)(nil), (*MaasMachineTemplateResource)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasMachineTemplateResource_To_v1alpha3_MaasMachineTemplateResource(a.(*v1alpha4.MaasMachineTemplateResource), b.(*MaasMachineTemplateResource), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MaasMachineTemplateSpec)(nil), (*v1alpha4.MaasMachineTemplateSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasMachineTemplateSpec_To_v1alpha4_MaasMachineTemplateSpec(a.(*MaasMachineTemplateSpec), b.(*v1alpha4.MaasMachineTemplateSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.MaasMachineTemplateSpec)(nil), (*MaasMachineTemplateSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasMachineTemplateSpec_To_v1alpha3_MaasMachineTemplateSpec(a.(*v1alpha4.MaasMachineTemplateSpec), b.(*MaasMachineTemplateSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Machine)(nil), (*v1alpha4.Machine)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_Machine_To_v1alpha4_Machine(a.(*Machine), b.(*v1alpha4.Machine), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.Machine)(nil), (*Machine)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_Machine_To_v1alpha3_Machine(a.(*v1alpha4.Machine), b.(*Machine), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Network)(nil), (*v1alpha4.Network)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_Network_To_v1alpha4_Network(a.(*Network), b.(*v1alpha4.Network), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha4.Network)(nil), (*Network)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_Network_To_v1alpha3_Network(a.(*v1alpha4.Network), b.(*Network), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*MaasMachineSpec)(nil), (*v1alpha4.MaasMachineSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha3_MaasMachineSpec_To_v1alpha4_MaasMachineSpec(a.(*MaasMachineSpec), b.(*v1alpha4.MaasMachineSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*v1alpha4.MaasMachineSpec)(nil), (*MaasMachineSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha4_MaasMachineSpec_To_v1alpha3_MaasMachineSpec(a.(*v1alpha4.MaasMachineSpec), b.(*MaasMachineSpec), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha3_APIEndpoint_To_v1alpha4_APIEndpoint(in *APIEndpoint, out *v1alpha4.APIEndpoint, s conversion.Scope) error {
	out.Host = in.Host
	out.Port = in.Port
	return nil
}

// Convert_v1alpha3_APIEndpoint_To_v1alpha4_APIEndpoint is an autogenerated conversion function.
func Convert_v1alpha3_APIEndpoint_To_v1alpha4_APIEndpoint(in *APIEndpoint, out *v1alpha4.APIEndpoint, s conversion.Scope) error {
	return autoConvert_v1alpha3_APIEndpoint_To_v1alpha4_APIEndpoint(in, out, s)
}

func autoConvert_v1alpha4_APIEndpoint_To_v1alpha3_APIEndpoint(in *v1alpha4.APIEndpoint, out *APIEndpoint, s conversion.Scope) error {
	out.Host = in.Host
	out.Port = in.Port
	return nil
}

// Convert_v1alpha4_APIEndpoint_To_v1alpha3_APIEndpoint is an autogenerated conversion function.
func Convert_v1alpha4_APIEndpoint_To_v1alpha3_APIEndpoint(in *v1alpha4.APIEndpoint, out *APIEndpoint, s conversion.Scope) error {
	return autoConvert_v1alpha4_APIEndpoint_To_v1alpha3_APIEndpoint(in, out, s)
}

func autoConvert_v1alpha3_MaasCluster_To_v1alpha4_MaasCluster(in *MaasCluster, out *v1alpha4.MaasCluster, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha3_MaasClusterSpec_To_v1alpha4_MaasClusterSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_v1alpha3_MaasClusterStatus_To_v1alpha4_MaasClusterStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha3_MaasCluster_To_v1alpha4_MaasCluster is an autogenerated conversion function.
func Convert_v1alpha3_MaasCluster_To_v1alpha4_MaasCluster(in *MaasCluster, out *v1alpha4.MaasCluster, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasCluster_To_v1alpha4_MaasCluster(in, out, s)
}

func autoConvert_v1alpha4_MaasCluster_To_v1alpha3_MaasCluster(in *v1alpha4.MaasCluster, out *MaasCluster, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha4_MaasClusterSpec_To_v1alpha3_MaasClusterSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_v1alpha4_MaasClusterStatus_To_v1alpha3_MaasClusterStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha4_MaasCluster_To_v1alpha3_MaasCluster is an autogenerated conversion function.
func Convert_v1alpha4_MaasCluster_To_v1alpha3_MaasCluster(in *v1alpha4.MaasCluster, out *MaasCluster, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasCluster_To_v1alpha3_MaasCluster(in, out, s)
}

func autoConvert_v1alpha3_MaasClusterList_To_v1alpha4_MaasClusterList(in *MaasClusterList, out *v1alpha4.MaasClusterList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]v1alpha4.MaasCluster)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v1alpha3_MaasClusterList_To_v1alpha4_MaasClusterList is an autogenerated conversion function.
func Convert_v1alpha3_MaasClusterList_To_v1alpha4_MaasClusterList(in *MaasClusterList, out *v1alpha4.MaasClusterList, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasClusterList_To_v1alpha4_MaasClusterList(in, out, s)
}

func autoConvert_v1alpha4_MaasClusterList_To_v1alpha3_MaasClusterList(in *v1alpha4.MaasClusterList, out *MaasClusterList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]MaasCluster)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v1alpha4_MaasClusterList_To_v1alpha3_MaasClusterList is an autogenerated conversion function.
func Convert_v1alpha4_MaasClusterList_To_v1alpha3_MaasClusterList(in *v1alpha4.MaasClusterList, out *MaasClusterList, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasClusterList_To_v1alpha3_MaasClusterList(in, out, s)
}

func autoConvert_v1alpha3_MaasClusterSpec_To_v1alpha4_MaasClusterSpec(in *MaasClusterSpec, out *v1alpha4.MaasClusterSpec, s conversion.Scope) error {
	out.DNSDomain = in.DNSDomain
	if err := Convert_v1alpha3_APIEndpoint_To_v1alpha4_APIEndpoint(&in.ControlPlaneEndpoint, &out.ControlPlaneEndpoint, s); err != nil {
		return err
	}
	out.FailureDomains = *(*[]string)(unsafe.Pointer(&in.FailureDomains))
	return nil
}

// Convert_v1alpha3_MaasClusterSpec_To_v1alpha4_MaasClusterSpec is an autogenerated conversion function.
func Convert_v1alpha3_MaasClusterSpec_To_v1alpha4_MaasClusterSpec(in *MaasClusterSpec, out *v1alpha4.MaasClusterSpec, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasClusterSpec_To_v1alpha4_MaasClusterSpec(in, out, s)
}

func autoConvert_v1alpha4_MaasClusterSpec_To_v1alpha3_MaasClusterSpec(in *v1alpha4.MaasClusterSpec, out *MaasClusterSpec, s conversion.Scope) error {
	out.DNSDomain = in.DNSDomain
	if err := Convert_v1alpha4_APIEndpoint_To_v1alpha3_APIEndpoint(&in.ControlPlaneEndpoint, &out.ControlPlaneEndpoint, s); err != nil {
		return err
	}
	out.FailureDomains = *(*[]string)(unsafe.Pointer(&in.FailureDomains))
	return nil
}

// Convert_v1alpha4_MaasClusterSpec_To_v1alpha3_MaasClusterSpec is an autogenerated conversion function.
func Convert_v1alpha4_MaasClusterSpec_To_v1alpha3_MaasClusterSpec(in *v1alpha4.MaasClusterSpec, out *MaasClusterSpec, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasClusterSpec_To_v1alpha3_MaasClusterSpec(in, out, s)
}

func autoConvert_v1alpha3_MaasClusterStatus_To_v1alpha4_MaasClusterStatus(in *MaasClusterStatus, out *v1alpha4.MaasClusterStatus, s conversion.Scope) error {
	out.Ready = in.Ready
	if err := Convert_v1alpha3_Network_To_v1alpha4_Network(&in.Network, &out.Network, s); err != nil {
		return err
	}
	out.FailureDomains = *(*apiv1alpha4.FailureDomains)(unsafe.Pointer(&in.FailureDomains))
	out.Conditions = *(*apiv1alpha4.Conditions)(unsafe.Pointer(&in.Conditions))
	return nil
}

// Convert_v1alpha3_MaasClusterStatus_To_v1alpha4_MaasClusterStatus is an autogenerated conversion function.
func Convert_v1alpha3_MaasClusterStatus_To_v1alpha4_MaasClusterStatus(in *MaasClusterStatus, out *v1alpha4.MaasClusterStatus, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasClusterStatus_To_v1alpha4_MaasClusterStatus(in, out, s)
}

func autoConvert_v1alpha4_MaasClusterStatus_To_v1alpha3_MaasClusterStatus(in *v1alpha4.MaasClusterStatus, out *MaasClusterStatus, s conversion.Scope) error {
	out.Ready = in.Ready
	if err := Convert_v1alpha4_Network_To_v1alpha3_Network(&in.Network, &out.Network, s); err != nil {
		return err
	}
	out.FailureDomains = *(*apiv1alpha3.FailureDomains)(unsafe.Pointer(&in.FailureDomains))
	out.Conditions = *(*apiv1alpha3.Conditions)(unsafe.Pointer(&in.Conditions))
	return nil
}

// Convert_v1alpha4_MaasClusterStatus_To_v1alpha3_MaasClusterStatus is an autogenerated conversion function.
func Convert_v1alpha4_MaasClusterStatus_To_v1alpha3_MaasClusterStatus(in *v1alpha4.MaasClusterStatus, out *MaasClusterStatus, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasClusterStatus_To_v1alpha3_MaasClusterStatus(in, out, s)
}

func autoConvert_v1alpha3_MaasMachine_To_v1alpha4_MaasMachine(in *MaasMachine, out *v1alpha4.MaasMachine, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha3_MaasMachineSpec_To_v1alpha4_MaasMachineSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_v1alpha3_MaasMachineStatus_To_v1alpha4_MaasMachineStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha3_MaasMachine_To_v1alpha4_MaasMachine is an autogenerated conversion function.
func Convert_v1alpha3_MaasMachine_To_v1alpha4_MaasMachine(in *MaasMachine, out *v1alpha4.MaasMachine, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasMachine_To_v1alpha4_MaasMachine(in, out, s)
}

func autoConvert_v1alpha4_MaasMachine_To_v1alpha3_MaasMachine(in *v1alpha4.MaasMachine, out *MaasMachine, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha4_MaasMachineSpec_To_v1alpha3_MaasMachineSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_v1alpha4_MaasMachineStatus_To_v1alpha3_MaasMachineStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha4_MaasMachine_To_v1alpha3_MaasMachine is an autogenerated conversion function.
func Convert_v1alpha4_MaasMachine_To_v1alpha3_MaasMachine(in *v1alpha4.MaasMachine, out *MaasMachine, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasMachine_To_v1alpha3_MaasMachine(in, out, s)
}

func autoConvert_v1alpha3_MaasMachineList_To_v1alpha4_MaasMachineList(in *MaasMachineList, out *v1alpha4.MaasMachineList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]v1alpha4.MaasMachine, len(*in))
		for i := range *in {
			if err := Convert_v1alpha3_MaasMachine_To_v1alpha4_MaasMachine(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_v1alpha3_MaasMachineList_To_v1alpha4_MaasMachineList is an autogenerated conversion function.
func Convert_v1alpha3_MaasMachineList_To_v1alpha4_MaasMachineList(in *MaasMachineList, out *v1alpha4.MaasMachineList, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasMachineList_To_v1alpha4_MaasMachineList(in, out, s)
}

func autoConvert_v1alpha4_MaasMachineList_To_v1alpha3_MaasMachineList(in *v1alpha4.MaasMachineList, out *MaasMachineList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MaasMachine, len(*in))
		for i := range *in {
			if err := Convert_v1alpha4_MaasMachine_To_v1alpha3_MaasMachine(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_v1alpha4_MaasMachineList_To_v1alpha3_MaasMachineList is an autogenerated conversion function.
func Convert_v1alpha4_MaasMachineList_To_v1alpha3_MaasMachineList(in *v1alpha4.MaasMachineList, out *MaasMachineList, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasMachineList_To_v1alpha3_MaasMachineList(in, out, s)
}

func autoConvert_v1alpha3_MaasMachineSpec_To_v1alpha4_MaasMachineSpec(in *MaasMachineSpec, out *v1alpha4.MaasMachineSpec, s conversion.Scope) error {
	out.FailureDomain = (*string)(unsafe.Pointer(in.FailureDomain))
	out.SystemID = (*string)(unsafe.Pointer(in.SystemID))
	out.ProviderID = (*string)(unsafe.Pointer(in.ProviderID))
	out.ResourcePool = (*string)(unsafe.Pointer(in.ResourcePool))
	// WARNING: in.MinCPU requires manual conversion: inconvertible types (*int vs int)
	// WARNING: in.MinMemory requires manual conversion: does not exist in peer-type
	out.Image = in.Image
	return nil
}

func autoConvert_v1alpha4_MaasMachineSpec_To_v1alpha3_MaasMachineSpec(in *v1alpha4.MaasMachineSpec, out *MaasMachineSpec, s conversion.Scope) error {
	out.FailureDomain = (*string)(unsafe.Pointer(in.FailureDomain))
	out.SystemID = (*string)(unsafe.Pointer(in.SystemID))
	out.ProviderID = (*string)(unsafe.Pointer(in.ProviderID))
	out.ResourcePool = (*string)(unsafe.Pointer(in.ResourcePool))
	// WARNING: in.MinCPU requires manual conversion: inconvertible types (int vs *int)
	// WARNING: in.MinMemoryInMB requires manual conversion: does not exist in peer-type
	out.Image = in.Image
	return nil
}

func autoConvert_v1alpha3_MaasMachineStatus_To_v1alpha4_MaasMachineStatus(in *MaasMachineStatus, out *v1alpha4.MaasMachineStatus, s conversion.Scope) error {
	out.Ready = in.Ready
	out.MachineState = (*v1alpha4.MachineState)(unsafe.Pointer(in.MachineState))
	out.MachinePowered = in.MachinePowered
	out.Hostname = (*string)(unsafe.Pointer(in.Hostname))
	out.DNSAttached = in.DNSAttached
	out.Addresses = *(*[]apiv1alpha4.MachineAddress)(unsafe.Pointer(&in.Addresses))
	out.Conditions = *(*apiv1alpha4.Conditions)(unsafe.Pointer(&in.Conditions))
	out.FailureReason = (*errors.MachineStatusError)(unsafe.Pointer(in.FailureReason))
	out.FailureMessage = (*string)(unsafe.Pointer(in.FailureMessage))
	return nil
}

// Convert_v1alpha3_MaasMachineStatus_To_v1alpha4_MaasMachineStatus is an autogenerated conversion function.
func Convert_v1alpha3_MaasMachineStatus_To_v1alpha4_MaasMachineStatus(in *MaasMachineStatus, out *v1alpha4.MaasMachineStatus, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasMachineStatus_To_v1alpha4_MaasMachineStatus(in, out, s)
}

func autoConvert_v1alpha4_MaasMachineStatus_To_v1alpha3_MaasMachineStatus(in *v1alpha4.MaasMachineStatus, out *MaasMachineStatus, s conversion.Scope) error {
	out.Ready = in.Ready
	out.MachineState = (*MachineState)(unsafe.Pointer(in.MachineState))
	out.MachinePowered = in.MachinePowered
	out.Hostname = (*string)(unsafe.Pointer(in.Hostname))
	out.DNSAttached = in.DNSAttached
	out.Addresses = *(*[]apiv1alpha3.MachineAddress)(unsafe.Pointer(&in.Addresses))
	out.Conditions = *(*apiv1alpha3.Conditions)(unsafe.Pointer(&in.Conditions))
	out.FailureReason = (*errors.MachineStatusError)(unsafe.Pointer(in.FailureReason))
	out.FailureMessage = (*string)(unsafe.Pointer(in.FailureMessage))
	return nil
}

// Convert_v1alpha4_MaasMachineStatus_To_v1alpha3_MaasMachineStatus is an autogenerated conversion function.
func Convert_v1alpha4_MaasMachineStatus_To_v1alpha3_MaasMachineStatus(in *v1alpha4.MaasMachineStatus, out *MaasMachineStatus, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasMachineStatus_To_v1alpha3_MaasMachineStatus(in, out, s)
}

func autoConvert_v1alpha3_MaasMachineTemplate_To_v1alpha4_MaasMachineTemplate(in *MaasMachineTemplate, out *v1alpha4.MaasMachineTemplate, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha3_MaasMachineTemplateSpec_To_v1alpha4_MaasMachineTemplateSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha3_MaasMachineTemplate_To_v1alpha4_MaasMachineTemplate is an autogenerated conversion function.
func Convert_v1alpha3_MaasMachineTemplate_To_v1alpha4_MaasMachineTemplate(in *MaasMachineTemplate, out *v1alpha4.MaasMachineTemplate, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasMachineTemplate_To_v1alpha4_MaasMachineTemplate(in, out, s)
}

func autoConvert_v1alpha4_MaasMachineTemplate_To_v1alpha3_MaasMachineTemplate(in *v1alpha4.MaasMachineTemplate, out *MaasMachineTemplate, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha4_MaasMachineTemplateSpec_To_v1alpha3_MaasMachineTemplateSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha4_MaasMachineTemplate_To_v1alpha3_MaasMachineTemplate is an autogenerated conversion function.
func Convert_v1alpha4_MaasMachineTemplate_To_v1alpha3_MaasMachineTemplate(in *v1alpha4.MaasMachineTemplate, out *MaasMachineTemplate, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasMachineTemplate_To_v1alpha3_MaasMachineTemplate(in, out, s)
}

func autoConvert_v1alpha3_MaasMachineTemplateList_To_v1alpha4_MaasMachineTemplateList(in *MaasMachineTemplateList, out *v1alpha4.MaasMachineTemplateList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]v1alpha4.MaasMachineTemplate, len(*in))
		for i := range *in {
			if err := Convert_v1alpha3_MaasMachineTemplate_To_v1alpha4_MaasMachineTemplate(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_v1alpha3_MaasMachineTemplateList_To_v1alpha4_MaasMachineTemplateList is an autogenerated conversion function.
func Convert_v1alpha3_MaasMachineTemplateList_To_v1alpha4_MaasMachineTemplateList(in *MaasMachineTemplateList, out *v1alpha4.MaasMachineTemplateList, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasMachineTemplateList_To_v1alpha4_MaasMachineTemplateList(in, out, s)
}

func autoConvert_v1alpha4_MaasMachineTemplateList_To_v1alpha3_MaasMachineTemplateList(in *v1alpha4.MaasMachineTemplateList, out *MaasMachineTemplateList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MaasMachineTemplate, len(*in))
		for i := range *in {
			if err := Convert_v1alpha4_MaasMachineTemplate_To_v1alpha3_MaasMachineTemplate(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_v1alpha4_MaasMachineTemplateList_To_v1alpha3_MaasMachineTemplateList is an autogenerated conversion function.
func Convert_v1alpha4_MaasMachineTemplateList_To_v1alpha3_MaasMachineTemplateList(in *v1alpha4.MaasMachineTemplateList, out *MaasMachineTemplateList, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasMachineTemplateList_To_v1alpha3_MaasMachineTemplateList(in, out, s)
}

func autoConvert_v1alpha3_MaasMachineTemplateResource_To_v1alpha4_MaasMachineTemplateResource(in *MaasMachineTemplateResource, out *v1alpha4.MaasMachineTemplateResource, s conversion.Scope) error {
	if err := Convert_v1alpha3_MaasMachineSpec_To_v1alpha4_MaasMachineSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha3_MaasMachineTemplateResource_To_v1alpha4_MaasMachineTemplateResource is an autogenerated conversion function.
func Convert_v1alpha3_MaasMachineTemplateResource_To_v1alpha4_MaasMachineTemplateResource(in *MaasMachineTemplateResource, out *v1alpha4.MaasMachineTemplateResource, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasMachineTemplateResource_To_v1alpha4_MaasMachineTemplateResource(in, out, s)
}

func autoConvert_v1alpha4_MaasMachineTemplateResource_To_v1alpha3_MaasMachineTemplateResource(in *v1alpha4.MaasMachineTemplateResource, out *MaasMachineTemplateResource, s conversion.Scope) error {
	if err := Convert_v1alpha4_MaasMachineSpec_To_v1alpha3_MaasMachineSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha4_MaasMachineTemplateResource_To_v1alpha3_MaasMachineTemplateResource is an autogenerated conversion function.
func Convert_v1alpha4_MaasMachineTemplateResource_To_v1alpha3_MaasMachineTemplateResource(in *v1alpha4.MaasMachineTemplateResource, out *MaasMachineTemplateResource, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasMachineTemplateResource_To_v1alpha3_MaasMachineTemplateResource(in, out, s)
}

func autoConvert_v1alpha3_MaasMachineTemplateSpec_To_v1alpha4_MaasMachineTemplateSpec(in *MaasMachineTemplateSpec, out *v1alpha4.MaasMachineTemplateSpec, s conversion.Scope) error {
	if err := Convert_v1alpha3_MaasMachineTemplateResource_To_v1alpha4_MaasMachineTemplateResource(&in.Template, &out.Template, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha3_MaasMachineTemplateSpec_To_v1alpha4_MaasMachineTemplateSpec is an autogenerated conversion function.
func Convert_v1alpha3_MaasMachineTemplateSpec_To_v1alpha4_MaasMachineTemplateSpec(in *MaasMachineTemplateSpec, out *v1alpha4.MaasMachineTemplateSpec, s conversion.Scope) error {
	return autoConvert_v1alpha3_MaasMachineTemplateSpec_To_v1alpha4_MaasMachineTemplateSpec(in, out, s)
}

func autoConvert_v1alpha4_MaasMachineTemplateSpec_To_v1alpha3_MaasMachineTemplateSpec(in *v1alpha4.MaasMachineTemplateSpec, out *MaasMachineTemplateSpec, s conversion.Scope) error {
	if err := Convert_v1alpha4_MaasMachineTemplateResource_To_v1alpha3_MaasMachineTemplateResource(&in.Template, &out.Template, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha4_MaasMachineTemplateSpec_To_v1alpha3_MaasMachineTemplateSpec is an autogenerated conversion function.
func Convert_v1alpha4_MaasMachineTemplateSpec_To_v1alpha3_MaasMachineTemplateSpec(in *v1alpha4.MaasMachineTemplateSpec, out *MaasMachineTemplateSpec, s conversion.Scope) error {
	return autoConvert_v1alpha4_MaasMachineTemplateSpec_To_v1alpha3_MaasMachineTemplateSpec(in, out, s)
}

func autoConvert_v1alpha3_Machine_To_v1alpha4_Machine(in *Machine, out *v1alpha4.Machine, s conversion.Scope) error {
	out.ID = in.ID
	out.Hostname = in.Hostname
	out.State = v1alpha4.MachineState(in.State)
	out.Powered = in.Powered
	out.AvailabilityZone = in.AvailabilityZone
	out.Addresses = *(*[]apiv1alpha4.MachineAddress)(unsafe.Pointer(&in.Addresses))
	return nil
}

// Convert_v1alpha3_Machine_To_v1alpha4_Machine is an autogenerated conversion function.
func Convert_v1alpha3_Machine_To_v1alpha4_Machine(in *Machine, out *v1alpha4.Machine, s conversion.Scope) error {
	return autoConvert_v1alpha3_Machine_To_v1alpha4_Machine(in, out, s)
}

func autoConvert_v1alpha4_Machine_To_v1alpha3_Machine(in *v1alpha4.Machine, out *Machine, s conversion.Scope) error {
	out.ID = in.ID
	out.Hostname = in.Hostname
	out.State = MachineState(in.State)
	out.Powered = in.Powered
	out.AvailabilityZone = in.AvailabilityZone
	out.Addresses = *(*[]apiv1alpha3.MachineAddress)(unsafe.Pointer(&in.Addresses))
	return nil
}

// Convert_v1alpha4_Machine_To_v1alpha3_Machine is an autogenerated conversion function.
func Convert_v1alpha4_Machine_To_v1alpha3_Machine(in *v1alpha4.Machine, out *Machine, s conversion.Scope) error {
	return autoConvert_v1alpha4_Machine_To_v1alpha3_Machine(in, out, s)
}

func autoConvert_v1alpha3_Network_To_v1alpha4_Network(in *Network, out *v1alpha4.Network, s conversion.Scope) error {
	out.DNSName = in.DNSName
	return nil
}

// Convert_v1alpha3_Network_To_v1alpha4_Network is an autogenerated conversion function.
func Convert_v1alpha3_Network_To_v1alpha4_Network(in *Network, out *v1alpha4.Network, s conversion.Scope) error {
	return autoConvert_v1alpha3_Network_To_v1alpha4_Network(in, out, s)
}

func autoConvert_v1alpha4_Network_To_v1alpha3_Network(in *v1alpha4.Network, out *Network, s conversion.Scope) error {
	out.DNSName = in.DNSName
	return nil
}

// Convert_v1alpha4_Network_To_v1alpha3_Network is an autogenerated conversion function.
func Convert_v1alpha4_Network_To_v1alpha3_Network(in *v1alpha4.Network, out *Network, s conversion.Scope) error {
	return autoConvert_v1alpha4_Network_To_v1alpha3_Network(in, out, s)
}
