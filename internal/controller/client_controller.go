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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	KeycloakRealm  string
}

// +kubebuilder:rbac:groups=keycloak.pewty.fr,resources=clients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keycloak.pewty.fr,resources=clients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=keycloak.pewty.fr,resources=clients/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch

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
	if kcClient.Spec.SecretRef.Name == "" {
		err := fmt.Errorf("secretRef.name is required")
		logger.Error(err, "Invalid Client spec")
		return ctrl.Result{}, err
	}

	// Get client credentials from referenced secret
	clientID, clientSecret, err := r.getClientCredentials(ctx, &kcClient)
	if err != nil {
		logger.Error(err, "Failed to get client credentials from secret")
		r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "SecretReadFailed", fmt.Sprintf("Failed to read secret: %v", err))
		return ctrl.Result{}, err
	}

	// Authenticate with Keycloak
	token, err := r.KeycloakClient.LoginClient(ctx, r.KeycloakUser, r.KeycloakPass, r.KeycloakRealm)
	if err != nil {
		logger.Error(err, "Failed to authenticate with Keycloak")
		r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "AuthenticationFailed", fmt.Sprintf("Failed to authenticate: %v", err))
		return ctrl.Result{}, err
	}

	// 2. Handle deletion logic with finalizer
	if !kcClient.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&kcClient, clientFinalizer) {
			// Resource is being deleted, perform cleanup
			// Get clientID from secret before deletion
			deleteClientID, _, err := r.getClientCredentials(ctx, &kcClient)
			if err != nil {
				logger.Error(err, "Failed to get client credentials for deletion, skipping Keycloak cleanup")
				// Continue with finalizer removal even if we can't read the secret
			} else {
				if err := r.deleteClientInKeycloak(ctx, r.KeycloakClient, token.AccessToken, &kcClient, deleteClientID); err != nil {
					logger.Error(err, "Failed to delete client in Keycloak")
					r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "DeletionFailed", fmt.Sprintf("Failed to delete: %v", err))
					return ctrl.Result{}, err
				}
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
		ClientID: &clientID,
	})
	if err != nil {
		logger.Error(err, "Failed to query Keycloak clients")
		r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "QueryFailed", fmt.Sprintf("Failed to query clients: %v", err))
		return ctrl.Result{}, err
	}

	if len(clients) == 0 {
		// 5. Client doesn't exist, create it
		logger.Info("Creating client in Keycloak", "clientID", clientID)

		newClient := r.convertToGoCloak(&kcClient.Spec.Client, clientID, clientSecret)
		clientID, err := r.KeycloakClient.CreateClient(ctx, token.AccessToken, *kcClient.Spec.Realm, newClient)
		if err != nil {
			logger.Error(err, "Failed to create client in Keycloak")
			r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "CreationFailed", fmt.Sprintf("Failed to create: %v", err))
			return ctrl.Result{}, err
		}

		logger.Info("Successfully created client in Keycloak", "clientID", clientID, "id", clientID)

		// Get the created client to retrieve generated secret
		createdClient, err := r.KeycloakClient.GetClient(ctx, token.AccessToken, *kcClient.Spec.Realm, clientID)
		if err != nil {
			logger.Error(err, "Failed to get created client details")
			r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "CreationFailed", fmt.Sprintf("Client created but failed to retrieve: %v", err))
			return ctrl.Result{}, err
		}

		// Update secret with credentials
		if err := r.updateSecretWithCredentials(ctx, &kcClient, createdClient.ClientID, createdClient.Secret); err != nil {
			logger.Error(err, "Failed to update secret with credentials")
			r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "SecretUpdateFailed", fmt.Sprintf("Failed to update secret: %v", err))
			return ctrl.Result{}, err
		}

		r.updateStatus(ctx, &kcClient, metav1.ConditionTrue, "Created", "Client successfully created in Keycloak")
	} else {
		// 6. Client exists, update it
		logger.Info("Updating client in Keycloak", "clientID", clientID)

		existingClient := clients[0]
		updatedClient := r.convertToGoCloak(&kcClient.Spec.Client, clientID, clientSecret)

		// Preserve the internal ID from the existing client
		updatedClient.ID = existingClient.ID

		err := r.KeycloakClient.UpdateClient(ctx, token.AccessToken, *kcClient.Spec.Realm, updatedClient)
		if err != nil {
			logger.Error(err, "Failed to update client in Keycloak")
			r.updateStatus(ctx, &kcClient, metav1.ConditionFalse, "UpdateFailed", fmt.Sprintf("Failed to update: %v", err))
			return ctrl.Result{}, err
		}

		// Update secret with current credentials (in case secret was regenerated)
		if updatedClient.Secret != nil {
			if err := r.updateSecretWithCredentials(ctx, &kcClient, updatedClient.ClientID, updatedClient.Secret); err != nil {
				logger.Error(err, "Failed to update secret with credentials")
				// Don't fail the reconciliation for secret update failures
			}
		}

		logger.Info("Successfully updated client in Keycloak", "clientID", clientID)
		r.updateStatus(ctx, &kcClient, metav1.ConditionTrue, "Updated", "Client successfully updated in Keycloak")
	}

	return ctrl.Result{}, nil
}

// deleteClientInKeycloak deletes a client from Keycloak if it exists
func (r *ClientReconciler) deleteClientInKeycloak(ctx context.Context, gc *gocloak.GoCloak, token string, kcClient *keycloakv1.Client, clientID string) error {
	logger := logf.FromContext(ctx)

	clients, err := gc.GetClients(ctx, token, *kcClient.Spec.Realm, gocloak.GetClientsParams{
		ClientID: &clientID,
	})
	if err != nil {
		return fmt.Errorf("failed to query client: %w", err)
	}

	if len(clients) > 0 {
		err := gc.DeleteClient(ctx, token, *kcClient.Spec.Realm, *clients[0].ID)
		if err != nil {
			return fmt.Errorf("failed to delete client: %w", err)
		}
		logger.Info("Successfully deleted client from Keycloak", "clientID", clientID)
	} else {
		logger.Info("Client not found in Keycloak, nothing to delete", "clientID", clientID)
	}

	return nil
}

// convertToGoCloak converts the CRD ClientRepresentation to gocloak.Client
func (r *ClientReconciler) convertToGoCloak(clientRep *keycloakv1.ClientRepresentation, clientID string, clientSecret string) gocloak.Client {
	gc := gocloak.Client{
		ID:                                 clientRep.ID,
		ClientID:                           &clientID,
		Name:                               clientRep.Name,
		Description:                        clientRep.Description,
		RootURL:                            clientRep.RootURL,
		AdminURL:                           clientRep.AdminURL,
		BaseURL:                            clientRep.BaseURL,
		SurrogateAuthRequired:              clientRep.SurrogateAuthRequired,
		Enabled:                            clientRep.Enabled,
		ClientAuthenticatorType:            clientRep.ClientAuthenticatorType,
		Secret:                             &clientSecret,
		RegistrationAccessToken:            clientRep.RegistrationAccessToken,
		DefaultRoles:                       &clientRep.DefaultRoles,
		RedirectURIs:                       &clientRep.RedirectUris,
		WebOrigins:                         &clientRep.WebOrigins,
		NotBefore:                          clientRep.NotBefore,
		BearerOnly:                         clientRep.BearerOnly,
		ConsentRequired:                    clientRep.ConsentRequired,
		StandardFlowEnabled:                clientRep.StandardFlowEnabled,
		ImplicitFlowEnabled:                clientRep.ImplicitFlowEnabled,
		DirectAccessGrantsEnabled:          clientRep.DirectAccessGrantsEnabled,
		ServiceAccountsEnabled:             clientRep.ServiceAccountsEnabled,
		AuthorizationServicesEnabled:       clientRep.AuthorizationServicesEnabled,
		PublicClient:                       clientRep.PublicClient,
		FrontChannelLogout:                 clientRep.FrontchannelLogout,
		Protocol:                           clientRep.Protocol,
		Attributes:                         &clientRep.Attributes,
		AuthenticationFlowBindingOverrides: &clientRep.AuthenticationFlowBindingOverrides,
		FullScopeAllowed:                   clientRep.FullScopeAllowed,
		NodeReRegistrationTimeout:          clientRep.NodeReRegistrationTimeout,
		DefaultClientScopes:                &clientRep.DefaultClientScopes,
		OptionalClientScopes:               &clientRep.OptionalClientScopes,
		Origin:                             clientRep.Origin,
	}

	// Convert RegisteredNodes from map[string]int32 to *map[string]int
	if clientRep.RegisteredNodes != nil {
		registeredNodes := make(map[string]int)
		for k, v := range clientRep.RegisteredNodes {
			registeredNodes[k] = int(v)
		}
		gc.RegisteredNodes = &registeredNodes
	}

	// Convert Access from map[string]bool to *map[string]interface{}
	if clientRep.Access != nil {
		access := make(map[string]interface{})
		for k, v := range clientRep.Access {
			access[k] = v
		}
		gc.Access = &access
	}

	// Convert protocol mappers
	if len(clientRep.ProtocolMappers) > 0 {
		protocolMappers := make([]gocloak.ProtocolMapperRepresentation, len(clientRep.ProtocolMappers))
		for i, pm := range clientRep.ProtocolMappers {
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

// getClientCredentials reads clientID and clientSecret from the referenced Kubernetes Secret
func (r *ClientReconciler) getClientCredentials(ctx context.Context, kcClient *keycloakv1.Client) (string, string, error) {
	logger := logf.FromContext(ctx)

	// Default keys
	clientIDKey := "clientId"
	clientSecretKey := "clientSecret"

	if kcClient.Spec.SecretRef.ClientIDKey != "" {
		clientIDKey = kcClient.Spec.SecretRef.ClientIDKey
	}
	if kcClient.Spec.SecretRef.ClientSecretKey != "" {
		clientSecretKey = kcClient.Spec.SecretRef.ClientSecretKey
	}

	// Get the secret
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Name:      kcClient.Spec.SecretRef.Name,
		Namespace: kcClient.Namespace,
	}

	if err := r.Get(ctx, secretName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return "", "", fmt.Errorf("secret %s not found in namespace %s", kcClient.Spec.SecretRef.Name, kcClient.Namespace)
		}
		return "", "", fmt.Errorf("failed to get secret: %w", err)
	}

	// Read clientID
	clientIDBytes, ok := secret.Data[clientIDKey]
	if !ok || len(clientIDBytes) == 0 {
		return "", "", fmt.Errorf("key %s not found in secret %s", clientIDKey, kcClient.Spec.SecretRef.Name)
	}
	clientID := string(clientIDBytes)

	// Read clientSecret
	clientSecretBytes, ok := secret.Data[clientSecretKey]
	if !ok || len(clientSecretBytes) == 0 {
		logger.Info("Client secret not found in secret, will be generated by Keycloak", "secretKey", clientSecretKey)
		return clientID, "", nil
	}
	clientSecret := string(clientSecretBytes)

	return clientID, clientSecret, nil
}

// updateSecretWithCredentials updates the referenced Kubernetes Secret with client credentials
func (r *ClientReconciler) updateSecretWithCredentials(ctx context.Context, kcClient *keycloakv1.Client, clientID *string, clientSecret *string) error {
	logger := logf.FromContext(ctx)

	if clientID == nil || clientSecret == nil {
		logger.Info("Client credentials are nil, skipping secret update")
		return nil
	}

	// Default keys
	clientIDKey := "clientId"
	clientSecretKey := "clientSecret"

	if kcClient.Spec.SecretRef.ClientIDKey != "" {
		clientIDKey = kcClient.Spec.SecretRef.ClientIDKey
	}
	if kcClient.Spec.SecretRef.ClientSecretKey != "" {
		clientSecretKey = kcClient.Spec.SecretRef.ClientSecretKey
	}

	// Get the secret
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Name:      kcClient.Spec.SecretRef.Name,
		Namespace: kcClient.Namespace,
	}

	if err := r.Get(ctx, secretName, secret); err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	// Update secret data
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data[clientIDKey] = []byte(*clientID)
	secret.Data[clientSecretKey] = []byte(*clientSecret)

	if err := r.Update(ctx, secret); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	logger.Info("Successfully updated secret with client credentials", "secret", kcClient.Spec.SecretRef.Name)
	return nil
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
