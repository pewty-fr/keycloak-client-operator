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

	gocloak "github.com/Nerzal/gocloak/v13"
)

// GoCloak defines the Keycloak API methods used by the controller.
// This interface enables dependency injection and testing with mocks.
// The concrete *gocloak.GoCloak type satisfies this interface.
type GoCloak interface {
	LoginClient(ctx context.Context, clientID, clientSecret, realm string, scopes ...string) (*gocloak.JWT, error)
	GetClients(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error)
	CreateClient(ctx context.Context, accessToken, realm string, newClient gocloak.Client) (string, error)
	GetClient(ctx context.Context, accessToken, realm, idOfClient string) (*gocloak.Client, error)
	UpdateClient(ctx context.Context, accessToken, realm string, updatedClient gocloak.Client) error
	DeleteClient(ctx context.Context, accessToken, realm, idOfClient string) error
}
