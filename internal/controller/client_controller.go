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
	"os"

	"github.com/Nerzal/gocloak/v13"
	"github.com/rs/zerolog/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	keycloakv1 "pewty.fr/keycloak-client-operator/api/v1"
)

// ClientReconciler reconciles a Client object
type ClientReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *ClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// 1. Get the Client resource from Kubernetes
	var kcClient keycloakv1.Client
	if err := r.Get(ctx, req.NamespacedName, &kcClient); err != nil {
		if apierrors.IsNotFound(err) {
			// Resource not found, it might have been deleted
			log.Error().Err(err).Msgf("failed to delete client %s/%s in keycloak", req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	gc := gocloak.NewClient(os.Getenv("KEYCLOAK_URL"))
	token, _ := gc.LoginAdmin(ctx, os.Getenv("KEYCLOAK_USER"), os.Getenv("KEYCLOAK_PASSWORD"), *kcClient.Spec.Realm)
	// Handle deletion logic
	if kcClient.DeletionTimestamp != nil {
		// Resource is being deleted, you can access the previous state here
		// Perform any cleanup or finalization logic
		// 2. If not found in K8s, check if it exists in Keycloak and delete if so
		// (pseudo code, you need to implement gocloak client setup and authentication)
		clients, _ := gc.GetClients(ctx, token.AccessToken, *kcClient.Spec.Realm, gocloak.GetClientsParams{ClientID: kcClient.Spec.Client.ID})
		if len(clients) > 0 {
			err := gc.DeleteClient(ctx, token.AccessToken, *kcClient.Spec.Realm, *clients[0].ID)
			if err != nil {
				log.Error().Err(err).Msg("failed to delete client in keycloak")
				return ctrl.Result{}, err
			}
			log.Info().Str("clientID", *kcClient.Spec.Client.ID).Msg("deleted client in keycloak")
		}
	}

	// 3. If found in K8s, check if it exists in Keycloak
	clients, _ := gc.GetClients(ctx, token.AccessToken, *kcClient.Spec.Realm, gocloak.GetClientsParams{ClientID: kcClient.Spec.Client.ID})

	if len(clients) == 0 {
		// 4. If not exists in Keycloak, create it
		newClient := gocloak.Client{
			ClientID: kcClient.Spec.Client.ID,
			// ...map other fields from kcClient.Spec...
		}
		_, err := gc.CreateClient(ctx, token.AccessToken, *kcClient.Spec.Realm, newClient)
		if err != nil {
			log.Error().Err(err).Msg("failed to create client in keycloak")
			return ctrl.Result{}, err
		}
		log.Info().Str("clientID", *kcClient.Spec.Client.ID).Msg("created client in keycloak")
	} else {
		// 5. If exists, update it
		existingClient := clients[0]
		// ...update fields as needed...
		err := gc.UpdateClient(ctx, token.AccessToken, *kcClient.Spec.Realm, *existingClient)
		if err != nil {
			log.Error().Err(err).Msg("failed to update client in keycloak")
			return ctrl.Result{}, err
		}
		log.Info().Str("clientID", *kcClient.Spec.Client.ID).Msg("updated client in keycloak")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakv1.Client{}).
		Named("client").
		Complete(r)
}
