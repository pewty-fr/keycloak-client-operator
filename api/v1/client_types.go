/*
Copyright 2025.

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

// ClientSpec defines the desired state of Client
type ClientSpec struct {
	Realm  *string              `json:"realm"`
	Client ClientRepresentation `json:"client"`
}

type ClientRepresentation struct {
	ID                                 *string                        `json:"id,omitempty"`
	ClientID                           *string                        `json:"clientId,omitempty"`
	Name                               *string                        `json:"name,omitempty"`
	Description                        *string                        `json:"description,omitempty"`
	Type                               *string                        `json:"type,omitempty"`
	RootURL                            *string                        `json:"rootUrl,omitempty"`
	AdminURL                           *string                        `json:"adminUrl,omitempty"`
	BaseURL                            *string                        `json:"baseUrl,omitempty"`
	SurrogateAuthRequired              *bool                          `json:"surrogateAuthRequired,omitempty"`
	Enabled                            *bool                          `json:"enabled,omitempty"`
	AlwaysDisplayInConsole             *bool                          `json:"alwaysDisplayInConsole,omitempty"`
	ClientAuthenticatorType            *string                        `json:"clientAuthenticatorType,omitempty"`
	Secret                             *string                        `json:"secret,omitempty"`
	RegistrationAccessToken            *string                        `json:"registrationAccessToken,omitempty"`
	DefaultRoles                       []string                       `json:"defaultRoles,omitempty"`
	RedirectUris                       []string                       `json:"redirectUris,omitempty"`
	WebOrigins                         []string                       `json:"webOrigins,omitempty"`
	NotBefore                          *int32                         `json:"notBefore,omitempty"`
	BearerOnly                         *bool                          `json:"bearerOnly,omitempty"`
	ConsentRequired                    *bool                          `json:"consentRequired,omitempty"`
	StandardFlowEnabled                *bool                          `json:"standardFlowEnabled,omitempty"`
	ImplicitFlowEnabled                *bool                          `json:"implicitFlowEnabled,omitempty"`
	DirectAccessGrantsEnabled          *bool                          `json:"directAccessGrantsEnabled,omitempty"`
	ServiceAccountsEnabled             *bool                          `json:"serviceAccountsEnabled,omitempty"`
	AuthorizationServicesEnabled       *bool                          `json:"authorizationServicesEnabled,omitempty"`
	DirectGrantsOnly                   *bool                          `json:"directGrantsOnly,omitempty"`
	PublicClient                       *bool                          `json:"publicClient,omitempty"`
	FrontchannelLogout                 *bool                          `json:"frontchannelLogout,omitempty"`
	Protocol                           *string                        `json:"protocol,omitempty"`
	Attributes                         map[string]string              `json:"attributes,omitempty"`
	AuthenticationFlowBindingOverrides map[string]string              `json:"authenticationFlowBindingOverrides,omitempty"`
	FullScopeAllowed                   *bool                          `json:"fullScopeAllowed,omitempty"`
	NodeReRegistrationTimeout          *int32                         `json:"nodeReRegistrationTimeout,omitempty"`
	RegisteredNodes                    map[string]int32               `json:"registeredNodes,omitempty"`
	ProtocolMappers                    []ProtocolMapperRepresentation `json:"protocolMappers,omitempty"`
	ClientTemplate                     *string                        `json:"clientTemplate,omitempty"`
	UseTemplateConfig                  *bool                          `json:"useTemplateConfig,omitempty"`
	UseTemplateScope                   *bool                          `json:"useTemplateScope,omitempty"`
	UseTemplateMappers                 *bool                          `json:"useTemplateMappers,omitempty"`
	DefaultClientScopes                []string                       `json:"defaultClientScopes,omitempty"`
	OptionalClientScopes               []string                       `json:"optionalClientScopes,omitempty"`
	AuthorizationSettings              *ResourceServerRepresentation  `json:"authorizationSettings,omitempty"`
	Access                             map[string]bool                `json:"access,omitempty"`
	Origin                             *string                        `json:"origin,omitempty"`
}

// ProtocolMapperRepresentation represents a protocol mapper for a client.
type ProtocolMapperRepresentation struct {
	ID             *string           `json:"id,omitempty"`
	Name           *string           `json:"name,omitempty"`
	Protocol       *string           `json:"protocol,omitempty"`
	ProtocolMapper *string           `json:"protocolMapper,omitempty"`
	Config         map[string]string `json:"config,omitempty"`
}

// ResourceServerRepresentation represents the authorization settings for a client.
type ResourceServerRepresentation struct {
	ID                            *string                  `json:"id,omitempty"`
	ClientID                      *string                  `json:"clientId,omitempty"`
	Name                          *string                  `json:"name,omitempty"`
	AllowRemoteResourceManagement *bool                    `json:"allowRemoteResourceManagement,omitempty"`
	PolicyEnforcementMode         PolicyEnforcementMode    `json:"policyEnforcementMode,omitempty"`
	Resources                     []ResourceRepresentation `json:"resources,omitempty"`
	Policies                      []PolicyRepresentation   `json:"policies,omitempty"`
	Scopes                        []ScopeRepresentation    `json:"scopes,omitempty"`
	DecisionStrategy              DecisionStrategy         `json:"decisionStrategy,omitempty"`
	AuthorizationSchema           AuthorizationSchema      `json:"authorizationSchema,omitempty"`
}

// ResourceRepresentation represents a Keycloak resource.
type ResourceRepresentation struct {
	ID                 *string                      `json:"_id,omitempty"`
	Name               *string                      `json:"name,omitempty"`
	Uris               []string                     `json:"uris,omitempty"`
	Type               *string                      `json:"type,omitempty"`
	Scopes             []ScopeRepresentation        `json:"scopes,omitempty"`
	IconURI            *string                      `json:"icon_uri,omitempty"`
	Owner              *ResourceOwnerRepresentation `json:"owner,omitempty"`
	OwnerManagedAccess *bool                        `json:"ownerManagedAccess,omitempty"`
	DisplayName        *string                      `json:"displayName,omitempty"`
	Attributes         map[string][]string          `json:"attributes,omitempty"`
	Uri                string                       `json:"uri,omitempty"`
	ScopesUma          []ScopeRepresentation        `json:"scopesUma,omitempty"`
}

// ResourceOwnerRepresentation represents the owner of a resource.
type ResourceOwnerRepresentation struct {
	ID   *string `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

// PolicyRepresentation represents a Keycloak policy.
type PolicyRepresentation struct {
	ID               *string                  `json:"id,omitempty"`
	Name             *string                  `json:"name,omitempty"`
	Description      *string                  `json:"description,omitempty"`
	Type             *string                  `json:"type,omitempty"`
	Policies         []string                 `json:"policies,omitempty"`
	Resources        []string                 `json:"resources,omitempty"`
	Scopes           []string                 `json:"scopes,omitempty"`
	Logic            *string                  `json:"logic,omitempty"`
	DecisionStrategy *string                  `json:"decisionStrategy,omitempty"`
	Owner            *string                  `json:"owner,omitempty"`
	ResourceType     *string                  `json:"resourceType,omitempty"`
	ResourcesData    []ResourceRepresentation `json:"resourcesData,omitempty"`
	ScopesData       []ScopeRepresentation    `json:"scopesData,omitempty"`
	Config           map[string]string        `json:"config,omitempty"`
}

// ScopeRepresentation represents a Keycloak scope.
type ScopeRepresentation struct {
	ID          *string                  `json:"id,omitempty"`
	Name        *string                  `json:"name,omitempty"`
	IconURI     *string                  `json:"iconUri,omitempty"`
	Policies    []PolicyRepresentation   `json:"policies,omitempty"`
	Resources   []ResourceRepresentation `json:"resources,omitempty"`
	DisplayName *string                  `json:"displayName,omitempty"`
}

// AuthorizationSchema represents the AuthorizationSchema object in Keycloak.
type AuthorizationSchema struct {
	ResourceTypes map[string]ResourceType `json:"resourceTypes,omitempty"`
}

// ResourceType represents a resource type in the AuthorizationSchema.
type ResourceType struct {
	Type         *string             `json:"type,omitempty"`
	Scopes       []string            `json:"scopes,omitempty"`
	ScopeAliases map[string][]string `json:"scopeAliases,omitempty"`
	GroupType    *string             `json:"groupType,omitempty"`
}

// PolicyEnforcementMode represents the enforcement mode for Keycloak authorization policies.
type PolicyEnforcementMode string

const (
    PolicyEnforcementModeEnforcing  PolicyEnforcementMode = "ENFORCING"
    PolicyEnforcementModePermissive PolicyEnforcementMode = "PERMISSIVE"
    PolicyEnforcementModeDisabled   PolicyEnforcementMode = "DISABLED"
)

// DecisionStrategy represents the decision strategy for Keycloak authorization policies.
type DecisionStrategy string

const (
    DecisionStrategyAffirmative DecisionStrategy = "AFFIRMATIVE"
    DecisionStrategyUnanimous   DecisionStrategy = "UNANIMOUS"
    DecisionStrategyConsensus   DecisionStrategy = "CONSENSUS"
)

// ClientStatus defines the observed state of Client.
type ClientStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Client resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Client is the Schema for the clients API
type Client struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Client
	// +required
	Spec ClientSpec `json:"spec"`

	// status defines the observed state of Client
	// +optional
	Status ClientStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ClientList contains a list of Client
type ClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Client `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Client{}, &ClientList{})
}
