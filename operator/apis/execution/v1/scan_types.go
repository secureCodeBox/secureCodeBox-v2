/*
Copyright 2020 iteratec GmbH.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ScanSpec defines the desired state of Scan
type ScanSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ScanType string `json:"scanType,omitempty"`

	Parameters []string `json:"parameters,omitempty"`
}

// ScanStatus defines the observed state of Scan
type ScanStatus struct {
	State string `json:"state,omitempty"`
	// RawResultType determines which kind of ParseDefinition will be used to turn the raw results of the scanner into findings
	RawResultType string `json:"rawResultType,omitempty"`
	// RawResultFile Filename of the result file of the scanner. e.g. `nmap-result.xml`
	RawResultFile string `json:"rawResultFile,omitempty"`

	// FindingCount indicates how many findings were identified in total
	FindingCount uint64 `json:"findingCount,omitempty"`
	// FindingSeverities indicates the count of finding with the respective severity
	FindingSeverities FindingSeverities `json:"findingSeverities,omitempty"`
	// FindingCategories indicates the count of finding broken down by their categories
	FindingCategories map[string]uint64 `json:"findingCategories,omitempty"`
}

// FindingSeverities indicates the count of finding with the respective severity
type FindingSeverities struct {
	InformationalCount uint64 `json:"informationalCount,omitempty"`
	LowCount           uint64 `json:"lowCount,omitempty"`
	MediumCount        uint64 `json:"mediumCount,omitempty"`
	HighCount          uint64 `json:"highCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="UID",type=string,JSONPath=`.metadata.uid`,description="K8s Resource UID",priority=1
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.scanType`,description="Scan Type"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="Scan State"
// +kubebuilder:printcolumn:name="Findings",type=string,JSONPath=`.status.findingCount`,description="Total Finding Count"
// +kubebuilder:printcolumn:name="Parameters",type=string,JSONPath=`.spec.parameters`,description="Arguments passed to the Scanner",priority=1

// Scan is the Schema for the scans API
type Scan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScanSpec   `json:"spec,omitempty"`
	Status ScanStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ScanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Scan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Scan{}, &ScanList{})
}