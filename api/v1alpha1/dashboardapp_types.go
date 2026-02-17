package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DashboardAppSpec defines the desired state of DashboardApp
type DashboardAppSpec struct {
	// Name is the display name of the application
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// URL is the application URL
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Category groups the app in the dashboard
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=media;ai;productivity;development;admin
	Category string `json:"category"`

	// Icon is the raw SVG string for the app icon
	// +kubebuilder:validation:Required
	Icon string `json:"icon"`

	// Groups defines which LDAP/OIDC groups can see this app (OR logic)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Groups []string `json:"groups"`

	// Priority controls sort order within a category (lower = first)
	// +kubebuilder:default=100
	// +optional
	Priority int `json:"priority,omitempty"`
}

// DashboardAppStatus defines the observed state of DashboardApp
type DashboardAppStatus struct {
	// Ready indicates if the app has been synced to the ConfigMap
	Ready bool `json:"ready,omitempty"`

	// LastSyncedAt is the timestamp of the last successful sync
	LastSyncedAt *metav1.Time `json:"lastSyncedAt,omitempty"`

	// Conditions represent the current state of the DashboardApp
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=dapp
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Category",type=string,JSONPath=`.spec.category`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DashboardApp is the Schema for the dashboardapps API
type DashboardApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DashboardAppSpec   `json:"spec,omitempty"`
	Status DashboardAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DashboardAppList contains a list of DashboardApp
type DashboardAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DashboardApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DashboardApp{}, &DashboardAppList{})
}
