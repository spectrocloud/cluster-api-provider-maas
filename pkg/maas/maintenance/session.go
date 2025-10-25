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

import (
	"context"
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8suuid "k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Keys used in the session ConfigMap data
	cmKeyOpID        = "opId"
	cmKeyStatus      = "status"
	cmKeyStartedAt   = "sessionStartTime"
	cmKeyCurrentHost = "systemID"
	cmKeyProgress    = "progress"
	cmKeyActive      = "active"
	cmKeyAffected    = "affectedWLCClusters"
	cmKeyPending     = "pendingReadyVMReplacements"

	// Optional trigger keys to initiate a session
	cmKeyTriggerStart = "start"
	cmKeyTriggerHost  = "hostSystemID"
)

// LoadSession loads the session ConfigMap and returns parsed state if present.
func LoadSession(ctx context.Context, c client.Client, namespace string) (State, *corev1.ConfigMap, error) {

	key := types.NamespacedName{Namespace: namespace, Name: SessionCMName}
	cm := &corev1.ConfigMap{}
	if err := c.Get(ctx, key, cm); err != nil {
		if apierrs.IsNotFound(err) {
			return State{}, nil, nil
		}
		return State{}, nil, err
	}
	st := State{}
	if cm.Data == nil {
		return st, cm, nil
	}
	st.OpID = cm.Data[cmKeyOpID]
	st.Status = Status(cm.Data[cmKeyStatus])
	st.CurrentHost = cm.Data[cmKeyCurrentHost]
	if ts := cm.Data[cmKeyStartedAt]; ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			st.StartedAt = t
		}
	}
	return st, cm, nil
}

// SaveSession writes the given state back to the session ConfigMap.
func SaveSession(ctx context.Context, c client.Client, namespace string, st State) error {
	key := types.NamespacedName{Namespace: namespace, Name: SessionCMName}
	cm := &corev1.ConfigMap{}
	err := c.Get(ctx, key, cm)
	if apierrs.IsNotFound(err) {
		cm = &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: SessionCMName, Namespace: namespace}}
	} else if err != nil {
		return err
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[cmKeyOpID] = st.OpID
	cm.Data[cmKeyStatus] = string(st.Status)
	cm.Data[cmKeyCurrentHost] = st.CurrentHost
	cm.Data[cmKeyStartedAt] = st.StartedAt.UTC().Format(time.RFC3339)
	cm.Data[cmKeyActive] = "true"
	if cm.Data[cmKeyAffected] == "" {
		cm.Data[cmKeyAffected] = "[]"
	}
	if cm.Data[cmKeyPending] == "" {
		cm.Data[cmKeyPending] = "[]"
	}
	if cm.Data[cmKeyProgress] == "" {
		cm.Data[cmKeyProgress] = "{}"
	}
	if cm.UID == "" {
		return c.Create(ctx, cm)
	}
	return c.Update(ctx, cm)
}

// StartSession initializes a new session if none Active is present. It will
// generate a fresh opId and set status Active.
func StartSession(ctx context.Context, c client.Client, namespace, currentHost string) (State, error) {
	st, cm, err := LoadSession(ctx, c, namespace)
	if err != nil {
		return State{}, err
	}
	if cm != nil && st.Status == StatusActive {
		return st, nil
	}
	newID := string(k8suuid.NewUUID())
	st = State{
		OpID:        newID,
		Status:      StatusActive,
		StartedAt:   time.Now().UTC(),
		CurrentHost: currentHost,
	}
	if err := SaveSession(ctx, c, namespace, st); err != nil {
		return State{}, err
	}
	return st, nil
}

// ShouldStartFromTrigger inspects optional trigger fields in the session CM.
func ShouldStartFromTrigger(cm *corev1.ConfigMap) (bool, string) {
	if cm == nil || cm.Data == nil {
		return false, ""
	}
	if cm.Data[cmKeyTriggerStart] == "true" {
		return true, cm.Data[cmKeyTriggerHost]
	}
	return false, ""
}

// UpdateProgress stores arbitrary progress map as JSON in the session CM.
func UpdateProgress(cm *corev1.ConfigMap, progress map[string]string) {
	if cm == nil {
		return
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	if b, err := json.Marshal(progress); err == nil {
		cm.Data[cmKeyProgress] = string(b)
	}
}
