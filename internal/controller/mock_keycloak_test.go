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
)

// mockGoCloak is a test double for the GoCloak interface.
// Set the *Func fields to override default behavior.
type mockGoCloak struct {
	LoginClientFunc  func(ctx context.Context, clientID, clientSecret, realm string, scopes ...string) (*gocloak.JWT, error)
	GetClientsFunc   func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error)
	CreateClientFunc func(ctx context.Context, accessToken, realm string, newClient gocloak.Client) (string, error)
	GetClientFunc    func(ctx context.Context, accessToken, realm, idOfClient string) (*gocloak.Client, error)
	UpdateClientFunc func(ctx context.Context, accessToken, realm string, updatedClient gocloak.Client) error
	DeleteClientFunc func(ctx context.Context, accessToken, realm, idOfClient string) error
}

func (m *mockGoCloak) LoginClient(ctx context.Context, clientID, clientSecret, realm string, scopes ...string) (*gocloak.JWT, error) {
	if m.LoginClientFunc != nil {
		return m.LoginClientFunc(ctx, clientID, clientSecret, realm, scopes...)
	}
	return &gocloak.JWT{AccessToken: "test-token"}, nil
}

func (m *mockGoCloak) GetClients(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
	if m.GetClientsFunc != nil {
		return m.GetClientsFunc(ctx, accessToken, realm, params)
	}
	return []*gocloak.Client{}, nil
}

func (m *mockGoCloak) CreateClient(ctx context.Context, accessToken, realm string, newClient gocloak.Client) (string, error) {
	if m.CreateClientFunc != nil {
		return m.CreateClientFunc(ctx, accessToken, realm, newClient)
	}
	return "default-internal-id", nil
}

func (m *mockGoCloak) GetClient(ctx context.Context, accessToken, realm, idOfClient string) (*gocloak.Client, error) {
	if m.GetClientFunc != nil {
		return m.GetClientFunc(ctx, accessToken, realm, idOfClient)
	}
	cid := "test-client"
	secret := "generated-secret"
	return &gocloak.Client{ClientID: &cid, Secret: &secret}, nil
}

func (m *mockGoCloak) UpdateClient(ctx context.Context, accessToken, realm string, updatedClient gocloak.Client) error {
	if m.UpdateClientFunc != nil {
		return m.UpdateClientFunc(ctx, accessToken, realm, updatedClient)
	}
	return nil
}

func (m *mockGoCloak) DeleteClient(ctx context.Context, accessToken, realm, idOfClient string) error {
	if m.DeleteClientFunc != nil {
		return m.DeleteClientFunc(ctx, accessToken, realm, idOfClient)
	}
	return nil
}

// errGoCloak returns a mockGoCloak where every call returns an error.
func errGoCloak(msg string) *mockGoCloak {
	err := fmt.Errorf("%s", msg) //nolint:goerr113
	return &mockGoCloak{
		LoginClientFunc:  func(_ context.Context, _, _, _ string, _ ...string) (*gocloak.JWT, error) { return nil, err },
		GetClientsFunc:   func(_ context.Context, _, _ string, _ gocloak.GetClientsParams) ([]*gocloak.Client, error) { return nil, err },
		CreateClientFunc: func(_ context.Context, _, _ string, _ gocloak.Client) (string, error) { return "", err },
		GetClientFunc:    func(_ context.Context, _, _, _ string) (*gocloak.Client, error) { return nil, err },
		UpdateClientFunc: func(_ context.Context, _, _ string, _ gocloak.Client) error { return err },
		DeleteClientFunc: func(_ context.Context, _, _, _ string) error { return err },
	}
}
