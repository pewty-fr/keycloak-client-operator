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

package controller

import (
	"context"
	"fmt"

	gocloak "github.com/Nerzal/gocloak/v13"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	keycloakv1 "github.com/pewty-fr/keycloak-client-operator/api/v1"
)

const clientFinalizer = "keycloak.pewty.fr/finalizer"

// ClientReconciler reconciles a Client object
type ClientReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	KeycloakClient *gocloak.GoCloak
	KeycloakURL    string
	KeycloakUser   string
	KeycloakPass   string
}

// +kubebuilder:rbac:groups=keycloak.pewty.fr,resources=clients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keycloak.pewty.fr,resources=clients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=keycloak.pewty.fr,resources=clients/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Client object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *ClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	// 1. Get the Client resource from Kubernetes
	var kcClient keycloakv1.Client
	if err := r.Get(ctx, req.NamespacedName, &kcClient); err != nil {
		if apierrors.IsNotFound(err) {
			// Resource not found, it has been deleted
			logger.Info("Client resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Client resource")
		return ctrl.Result{}, err
	}

	// Validate required fields
	if kcClient.Spec.Realm == nil {
		err := fmt.Errorf("realm is required")
		logger.Error(err, "Invalid Client spec")
		return ctrl.Result{}, err
	}
	if kcClient.Spec.Client.ClientID == nil {
		err := fmt.Errorf("clientId is required")
		logger.Error(err, "Invalid Client spec")
		return ctrl.Result{}, err
	}

	// Authenticate with Keycloak
	token, err := r.KeycloakClient.LoginAdmin(ctx, r.KeycloakUser, r.KeycloakPass, "master")
	if err != nil {
		logger.Error(err, "Failed to authenticate with Keycloak")
		r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "AuthenticationFailed", fmt.Sprintf("Failed to authenticate: %v", err))
		return ctrl.Result{}, err
	}

	// 2. Handle deletion logic with finalizer
	if !kcClient.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&kcClient, clientFinalizer) {
			// Resource is being deleted, perform cleanup
			if err := r.deleteClientInKeycloak(ctx, r.KeycloakClient, token.AccessToken, &kcClient); err != nil {
				logger.Error(err, "Failed to delete client in Keycloak")
				r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "DeletionFailed", fmt.Sprintf("Failed to delete: %v", err))
				return ctrl.Result{}, err
			}

			// Remove finalizer to allow deletion
			controllerutil.RemoveFinalizer(&kcClient, clientFinalizer)
			if err := r.Update(ctx, &kcClient); err != nil {
				logger.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 3. Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&kcClient, clientFinalizer) {
		controllerutil.AddFinalizer(&kcClient, clientFinalizer)
		if err := r.Update(ctx, &kcClient); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. Check if client exists in Keycloak
	clients, err := r.KeycloakClient.GetClients(ctx, token.AccessToken, *kcClient.Spec.Realm, gocloak.GetClientsParams{
		ClientID: kcClient.Spec.Client.ClientID,
	})
	if err != nil {
		logger.Error(err, "Failed to query Keycloak clients")
		r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "QueryFailed", fmt.Sprintf("Failed to query clients: %v", err))
		return ctrl.Result{}, err
	}

	if len(clients) == 0 {
		// 5. Client doesn't exist, create it
		logger.Info("Creating client in Keycloak", "clientID", *kcClient.Spec.Client.ClientID)

		newClient := r.convertToGoCloak(&kcClient.Spec.Client)
		clientID, err := r.KeycloakClient.CreateClient(ctx, token.AccessToken, *kcClient.Spec.Realm, newClient)
		if err != nil {
			logger.Error(err, "Failed to create client in Keycloak")
			r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "CreationFailed", fmt.Sprintf("Failed to create: %v", err))
			return ctrl.Result{}, err
		}

		logger.Info("Successfully created client in Keycloak", "clientID", *kcClient.Spec.Client.ClientID, "id", clientID)
		r.updateStatus(ctx, &kcClient, metav1.ConditionTrue, "Created", "Client successfully created in Keycloak")
	} else {
		// 6. Client exists, update it
		logger.Info("Updating client in Keycloak", "clientID", *kcClient.Spec.Client.ClientID)

		existingClient := clients[0]
		updatedClient := r.convertToGoCloak(&kcClient.Spec.Client)

		// Preserve the internal ID from the existing client
		updatedClient.ID = existingClient.ID

		err := r.KeycloakClient.UpdateClient(ctx, token.AccessToken, *kcClient.Spec.Realm, updatedClient)
		if err != nil {
			logger.Error(err, "Failed to update client in Keycloak")
			r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "UpdateFailed", fmt.Sprintf("Failed to update: %v", err))
			return ctrl.Result{}, err
		}

		logger.Info("Successfully updated client in Keycloak", "clientID", *kcClient.Spec.Client.ClientID)
		r.updateStatus(ctx, &kcClient, metav1.ConditionTrue, "Updated", "Client successfully updated in Keycloak")
	}

	return ctrl.Result{}, nil
}

// deleteClientInKeycloak deletes a client from Keycloak if it exists
func (r *ClientReconciler) deleteClientInKeycloak(ctx context.Context, gc *gocloak.GoCloak, token string, kcClient *keycloakv1.Client) error {
	logger := logf.FromContext(ctx)

	clients, err := gc.GetClients(ctx, token, *kcClient.Spec.Realm, gocloak.GetClientsParams{
		ClientID: kcClient.Spec.Client.ClientID,
	})
	if err != nil {
		return fmt.Errorf("failed to query client: %w", err)
	}

	if len(clients) > 0 {
		err := gc.DeleteClient(ctx, token, *kcClient.Spec.Realm, *clients[0].ID)
		if err != nil {
			return fmt.Errorf("failed to delete client: %w", err)
		}
		logger.Info("Successfully deleted client from Keycloak", "clientID", *kcClient.Spec.Client.ClientID)
	} else {
		logger.Info("Client not found in Keycloak, nothing to delete", "clientID", *kcClient.Spec.Client.ClientID)
	}

	return nil
}

// convertToGoCloak converts the CRD ClientRepresentation to gocloak.Client
func (r *ClientReconciler) convertToGoCloak(client *keycloakv1.ClientRepresentation) gocloak.Client {
	gc := gocloak.Client{
		ID:                                 client.ID,
		ClientID:                           client.ClientID,
		Name:                               client.Name,
		Description:                        client.Description,
		RootURL:                            client.RootURL,
		AdminURL:                           client.AdminURL,
		BaseURL:                            client.BaseURL,
		SurrogateAuthRequired:              client.SurrogateAuthRequired,
		Enabled:                            client.Enabled,
		ClientAuthenticatorType:            client.ClientAuthenticatorType,
		Secret:                             client.Secret,
		RegistrationAccessToken:            client.RegistrationAccessToken,
		DefaultRoles:                       &client.DefaultRoles,
		RedirectURIs:                       &client.RedirectUris,
		WebOrigins:                         &client.WebOrigins,
		NotBefore:                          client.NotBefore,
		BearerOnly:                         client.BearerOnly,
		ConsentRequired:                    client.ConsentRequired,
		StandardFlowEnabled:                client.StandardFlowEnabled,
		ImplicitFlowEnabled:                client.ImplicitFlowEnabled,
		DirectAccessGrantsEnabled:          client.DirectAccessGrantsEnabled,
		ServiceAccountsEnabled:             client.ServiceAccountsEnabled,
		AuthorizationServicesEnabled:       client.AuthorizationServicesEnabled,
		PublicClient:                       client.PublicClient,
		FrontChannelLogout:                 client.FrontchannelLogout,
		Protocol:                           client.Protocol,
		Attributes:                         &client.Attributes,
		AuthenticationFlowBindingOverrides: &client.AuthenticationFlowBindingOverrides,
		FullScopeAllowed:                   client.FullScopeAllowed,
		NodeReRegistrationTimeout:          client.NodeReRegistrationTimeout,
		DefaultClientScopes:                &client.DefaultClientScopes,
		OptionalClientScopes:               &client.OptionalClientScopes,
		Origin:                             client.Origin,
	}

	// Convert RegisteredNodes from map[string]int32 to *map[string]int
	if client.RegisteredNodes != nil {
		registeredNodes := make(map[string]int)
		for k, v := range client.RegisteredNodes {
			registeredNodes[k] = int(v)
		}
		gc.RegisteredNodes = &registeredNodes
	}

	// Convert Access from map[string]bool to *map[string]interface{}
	if client.Access != nil {
		access := make(map[string]interface{})
		for k, v := range client.Access {
			access[k] = v
		}
		gc.Access = &access
	}

	// Convert protocol mappers
	if len(client.ProtocolMappers) > 0 {
		protocolMappers := make([]gocloak.ProtocolMapperRepresentation, len(client.ProtocolMappers))
		for i, pm := range client.ProtocolMappers {
			protocolMappers[i] = gocloak.ProtocolMapperRepresentation{
				ID:             pm.ID,
				Name:           pm.Name,
				Protocol:       pm.Protocol,
				ProtocolMapper: pm.ProtocolMapper,
				Config:         &pm.Config,
			}
		}
		gc.ProtocolMappers = &protocolMappers
	}

	return gc
}

// updateStatus updates the Client resource status
func (r *ClientReconciler) updateStatus(ctx context.Context, kcClient *keycloakv1.Client, status metav1.ConditionStatus, reason, message string) {
	logger := logf.FromContext(ctx)

	condition := metav1.Condition{
		Type:               "Ready",
		Status:             status,
		ObservedGeneration: kcClient.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Update or append the condition
	updated := false
	for i, cond := range kcClient.Status.Conditions {
		if cond.Type == condition.Type {
			kcClient.Status.Conditions[i] = condition
			updated = true
			break
		}
	}
	if !updated {
		kcClient.Status.Conditions = append(kcClient.Status.Conditions, condition)
	}

	if err := r.Status().Update(ctx, kcClient); err != nil {
		logger.Error(err, "Failed to update Client status")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakv1.Client{}).
		Named("client").
		Complete(r)
}
