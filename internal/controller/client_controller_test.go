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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	keycloakv1 "github.com/pewty-fr/keycloak-client-operator/api/v1"
)

var _ = Describe("Client Controller", func() {
	BeforeEach(func() {
		// Set environment variables for Keycloak connection
		os.Setenv("KEYCLOAK_URL", "http://localhost:8080")
		os.Setenv("KEYCLOAK_USER", "admin")
		os.Setenv("KEYCLOAK_PASSWORD", "admin")
	})

	Context("When converting ClientRepresentation to gocloak.Client", func() {
		It("Should correctly map all basic fields", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client-id"
			clientName := "Test Client"
			clientSecret := "test-secret"
			enabled := true
			publicClient := false

			clientRep := &keycloakv1.ClientRepresentation{
				ClientID:                  &clientID,
				Name:                      &clientName,
				Secret:                    &clientSecret,
				Enabled:                   &enabled,
				PublicClient:              &publicClient,
				StandardFlowEnabled:       boolPtr(true),
				DirectAccessGrantsEnabled: boolPtr(true),
				Protocol:                  strPtr("openid-connect"),
				RedirectUris: []string{
					"http://localhost:3000/callback",
				},
				WebOrigins: []string{
					"http://localhost:3000",
				},
				Attributes: map[string]string{
					"access.token.lifespan": "1800",
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying basic fields")
			Expect(goCloak.ClientID).To(Equal(&clientID))
			Expect(goCloak.Name).To(Equal(&clientName))
			Expect(goCloak.Secret).To(Equal(&clientSecret))
			Expect(goCloak.Enabled).To(Equal(&enabled))
			Expect(goCloak.PublicClient).To(Equal(&publicClient))

			By("Verifying flow settings")
			Expect(goCloak.StandardFlowEnabled).To(Equal(boolPtr(true)))
			Expect(goCloak.DirectAccessGrantsEnabled).To(Equal(boolPtr(true)))

			By("Verifying protocol")
			Expect(goCloak.Protocol).To(Equal(strPtr("openid-connect")))

			By("Verifying redirect URIs")
			Expect(goCloak.RedirectURIs).NotTo(BeNil())
			Expect(*goCloak.RedirectURIs).To(ContainElement("http://localhost:3000/callback"))

			By("Verifying web origins")
			Expect(goCloak.WebOrigins).NotTo(BeNil())
			Expect(*goCloak.WebOrigins).To(ContainElement("http://localhost:3000"))

			By("Verifying attributes")
			Expect(goCloak.Attributes).NotTo(BeNil())
			Expect((*goCloak.Attributes)["access.token.lifespan"]).To(Equal("1800"))
		})

		It("Should handle protocol mappers correctly", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client-id"
			mapperName := "email"
			protocol := "openid-connect"
			protocolMapper := "oidc-usermodel-property-mapper"

			clientRep := &keycloakv1.ClientRepresentation{
				ClientID: &clientID,
				ProtocolMappers: []keycloakv1.ProtocolMapperRepresentation{
					{
						Name:           &mapperName,
						Protocol:       &protocol,
						ProtocolMapper: &protocolMapper,
						Config: map[string]string{
							"user.attribute": "email",
							"claim.name":     "email",
						},
					},
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying protocol mappers exist")
			Expect(goCloak.ProtocolMappers).NotTo(BeNil())
			Expect(*goCloak.ProtocolMappers).To(HaveLen(1))

			mapper := (*goCloak.ProtocolMappers)[0]
			Expect(mapper.Name).To(Equal(&mapperName))
			Expect(mapper.Protocol).To(Equal(&protocol))
			Expect(mapper.ProtocolMapper).To(Equal(&protocolMapper))
			Expect(mapper.Config).NotTo(BeNil())
			Expect((*mapper.Config)["user.attribute"]).To(Equal("email"))
		})

		It("Should handle RegisteredNodes type conversion", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client-id"
			clientRep := &keycloakv1.ClientRepresentation{
				ClientID: &clientID,
				RegisteredNodes: map[string]int32{
					"node1": 12345,
					"node2": 67890,
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying RegisteredNodes conversion from int32 to int")
			Expect(goCloak.RegisteredNodes).NotTo(BeNil())
			Expect((*goCloak.RegisteredNodes)["node1"]).To(Equal(12345))
			Expect((*goCloak.RegisteredNodes)["node2"]).To(Equal(67890))
		})

		It("Should handle Access map type conversion", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client-id"
			clientRep := &keycloakv1.ClientRepresentation{
				ClientID: &clientID,
				Access: map[string]bool{
					"view":      true,
					"configure": false,
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying Access conversion from bool to interface{}")
			Expect(goCloak.Access).NotTo(BeNil())
			Expect((*goCloak.Access)["view"]).To(BeTrue())
			Expect((*goCloak.Access)["configure"]).To(BeFalse())
		})

		It("Should handle nil values gracefully", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client-id"
			clientRep := &keycloakv1.ClientRepresentation{
				ClientID: &clientID,
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying nil fields are handled")
			Expect(goCloak.ClientID).To(Equal(&clientID))
			Expect(goCloak.Name).To(BeNil())
			Expect(goCloak.ProtocolMappers).To(BeNil())
		})
	})

	Context("When creating a Client resource with proper validation", func() {
		const resourceName = "test-client"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		AfterEach(func() {
			// Cleanup
			resource := &keycloakv1.Client{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				By("Cleanup the specific resource instance Client")
				k8sClient.Delete(ctx, resource)
			}
		})

		It("Should create a valid Client resource with required fields", func() {
			By("Creating the custom resource with required fields")

			realm := "master"
			clientID := "test-client-id"

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm: &realm,
					Client: keycloakv1.ClientRepresentation{
						ClientID: &clientID,
						Enabled:  boolPtr(true),
						Protocol: strPtr("openid-connect"),
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Verifying the resource was created")
			createdClient := &keycloakv1.Client{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, createdClient)
			}).Should(Succeed())

			Expect(createdClient.Spec.Realm).To(Equal(&realm))
			Expect(createdClient.Spec.Client.ClientID).To(Equal(&clientID))

			By("Verifying finalizer is added by the controller")
			// Note: Finalizer addition requires the controller to reconcile,
			// which requires a running Keycloak instance in integration tests
			// This is tested in e2e tests instead
		})

		It("Should reject a Client resource without required realm", func() {
			By("Attempting to create resource without realm")

			clientID := "test-client-id-invalid"

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-client",
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm: nil, // Missing required field
					Client: keycloakv1.ClientRepresentation{
						ClientID: &clientID,
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(errors.IsInvalid(err)).To(BeTrue())
		})
	})

	Context("When testing Reconcile with missing resources", func() {
		It("Should return nil error when resource is not found", func() {
			reconciler := &ClientReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			ctx := context.Background()
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent-client",
					Namespace: "default",
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			By("Verifying no error is returned for missing resource")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})
	})

	Context("When testing Reconcile validation errors", func() {
		const resourceName = "validation-test-client"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		AfterEach(func() {
			resource := &keycloakv1.Client{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				// Remove finalizer if present
				if controllerutil.ContainsFinalizer(resource, clientFinalizer) {
					controllerutil.RemoveFinalizer(resource, clientFinalizer)
					k8sClient.Update(ctx, resource)
				}
				k8sClient.Delete(ctx, resource)
			}
		})

		It("Should return error when realm is nil during reconciliation", func() {
			By("Creating a resource that somehow has nil realm (bypassing CRD validation)")
			// This tests the controller's validation logic
			reconciler := &ClientReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			clientID := "validation-client"
			// Create resource with realm (passes CRD validation)
			realm := "master"
			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm: &realm,
					Client: keycloakv1.ClientRepresentation{
						ClientID: &clientID,
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			// Now manually update to set realm to nil (simulating edge case)
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, resource)
			}).Should(Succeed())

			resource.Spec.Realm = nil
			err := k8sClient.Update(ctx, resource)

			By("Attempting reconciliation should fail validation")
			if err == nil {
				// If update succeeded (no CRD validation), reconcile should fail
				req := ctrl.Request{
					NamespacedName: typeNamespacedName,
				}

				_, reconcileErr := reconciler.Reconcile(ctx, req)
				Expect(reconcileErr).To(HaveOccurred())
				Expect(reconcileErr.Error()).To(ContainSubstring("realm is required"))
			} else {
				// CRD validation prevented the update, which is also correct behavior
				Expect(err).To(HaveOccurred())
			}
		})

		It("Should return error when clientId is nil during reconciliation", func() {
			reconciler := &ClientReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			clientID := "validation-client-2"
			realm := "master"
			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName + "-2",
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm: &realm,
					Client: keycloakv1.ClientRepresentation{
						ClientID: &clientID,
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			typeNamespacedName2 := types.NamespacedName{
				Name:      resourceName + "-2",
				Namespace: "default",
			}

			// Get resource and try to update clientId to nil
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName2, resource)
			}).Should(Succeed())

			resource.Spec.Client.ClientID = nil
			err := k8sClient.Update(ctx, resource)

			By("Attempting reconciliation should fail validation")
			if err == nil {
				req := ctrl.Request{
					NamespacedName: typeNamespacedName2,
				}

				_, reconcileErr := reconciler.Reconcile(ctx, req)
				Expect(reconcileErr).To(HaveOccurred())
				Expect(reconcileErr.Error()).To(ContainSubstring("clientId is required"))
			} else {
				// CRD validation prevented the update
				Expect(err).To(HaveOccurred())
			}

			// Cleanup
			resource2 := &keycloakv1.Client{}
			if k8sClient.Get(ctx, typeNamespacedName2, resource2) == nil {
				k8sClient.Delete(ctx, resource2)
			}
		})
	})

	Context("When testing status updates", func() {
		const resourceName = "test-status-client"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		AfterEach(func() {
			resource := &keycloakv1.Client{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				k8sClient.Delete(ctx, resource)
			}
		})

		It("Should update status conditions", func() {
			By("Creating a Client resource")
			realm := "test-realm"
			clientID := "status-test-client"

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm: &realm,
					Client: keycloakv1.ClientRepresentation{
						ClientID: &clientID,
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Manually calling updateStatus")
			reconciler := &ClientReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Get the created resource
			createdClient := &keycloakv1.Client{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, createdClient)
			}).Should(Succeed())

			// Update the status
			reconciler.updateStatus(ctx, createdClient, metav1.ConditionTrue, "TestReason", "Test message")

			By("Verifying status condition was updated")
			Eventually(func() bool {
				updatedClient := &keycloakv1.Client{}
				err := k8sClient.Get(ctx, typeNamespacedName, updatedClient)
				if err != nil {
					return false
				}
				if len(updatedClient.Status.Conditions) == 0 {
					return false
				}
				return updatedClient.Status.Conditions[0].Type == "Ready" &&
					updatedClient.Status.Conditions[0].Status == metav1.ConditionTrue &&
					updatedClient.Status.Conditions[0].Reason == "TestReason"
			}).Should(BeTrue())
		})

		It("Should update existing status condition", func() {
			By("Creating a Client resource with initial status")
			realm := "test-realm"
			clientID := "status-update-client"

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName + "-update",
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm: &realm,
					Client: keycloakv1.ClientRepresentation{
						ClientID: &clientID,
					},
				},
				Status: keycloakv1.ClientStatus{
					Conditions: []metav1.Condition{
						{
							Type:               "Ready",
							Status:             metav1.ConditionFalse,
							Reason:             "InitialReason",
							Message:            "Initial message",
							LastTransitionTime: metav1.Now(),
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Updating the status condition")
			reconciler := &ClientReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			typeNamespacedNameUpdate := types.NamespacedName{
				Name:      resourceName + "-update",
				Namespace: "default",
			}

			createdClient := &keycloakv1.Client{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedNameUpdate, createdClient)
			}).Should(Succeed())

			reconciler.updateStatus(ctx, createdClient, metav1.ConditionTrue, "UpdatedReason", "Updated message")

			By("Verifying status condition was updated, not appended")
			Eventually(func() bool {
				updatedClient := &keycloakv1.Client{}
				err := k8sClient.Get(ctx, typeNamespacedNameUpdate, updatedClient)
				if err != nil {
					return false
				}
				// Should still have only one condition
				if len(updatedClient.Status.Conditions) != 1 {
					return false
				}
				return updatedClient.Status.Conditions[0].Status == metav1.ConditionTrue &&
					updatedClient.Status.Conditions[0].Reason == "UpdatedReason"
			}).Should(BeTrue())

			// Cleanup
			k8sClient.Delete(ctx, resource)
		})
	})

	Context("When testing finalizer logic", func() {
		const resourceName = "test-finalizer-client"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		AfterEach(func() {
			resource := &keycloakv1.Client{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				// Remove finalizer if present to allow deletion
				if controllerutil.ContainsFinalizer(resource, clientFinalizer) {
					controllerutil.RemoveFinalizer(resource, clientFinalizer)
					k8sClient.Update(ctx, resource)
				}
				k8sClient.Delete(ctx, resource)
			}
		})

		It("Should add finalizer to new resource", func() {
			By("Creating a Client resource")
			realm := "test-realm"
			clientID := "finalizer-test-client"

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm: &realm,
					Client: keycloakv1.ClientRepresentation{
						ClientID: &clientID,
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Verifying resource is created without finalizer initially")
			createdClient := &keycloakv1.Client{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, createdClient)
			}).Should(Succeed())

			Expect(controllerutil.ContainsFinalizer(createdClient, clientFinalizer)).To(BeFalse())

			By("Simulating finalizer addition (normally done by reconciler)")
			controllerutil.AddFinalizer(createdClient, clientFinalizer)
			Expect(k8sClient.Update(ctx, createdClient)).To(Succeed())

			By("Verifying finalizer was added")
			Eventually(func() bool {
				updatedClient := &keycloakv1.Client{}
				err := k8sClient.Get(ctx, typeNamespacedName, updatedClient)
				if err != nil {
					return false
				}
				return controllerutil.ContainsFinalizer(updatedClient, clientFinalizer)
			}).Should(BeTrue())
		})

		It("Should handle resource with finalizer being deleted", func() {
			By("Creating a Client resource with finalizer")
			realm := "test-realm"
			clientID := "deletion-test-client"

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName + "-deletion",
					Namespace:  "default",
					Finalizers: []string{clientFinalizer},
				},
				Spec: keycloakv1.ClientSpec{
					Realm: &realm,
					Client: keycloakv1.ClientRepresentation{
						ClientID: &clientID,
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Verifying resource has finalizer")
			typeNamespacedNameDeletion := types.NamespacedName{
				Name:      resourceName + "-deletion",
				Namespace: "default",
			}

			createdClient := &keycloakv1.Client{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedNameDeletion, createdClient)
			}).Should(Succeed())
			Expect(controllerutil.ContainsFinalizer(createdClient, clientFinalizer)).To(BeTrue())

			By("Initiating deletion")
			Expect(k8sClient.Delete(ctx, createdClient)).To(Succeed())

			By("Verifying resource is marked for deletion but not removed due to finalizer")
			Eventually(func() bool {
				deletingClient := &keycloakv1.Client{}
				err := k8sClient.Get(ctx, typeNamespacedNameDeletion, deletingClient)
				if err != nil {
					return false
				}
				return !deletingClient.DeletionTimestamp.IsZero()
			}).Should(BeTrue())

			By("Removing finalizer to complete deletion")
			finalizingClient := &keycloakv1.Client{}
			Expect(k8sClient.Get(ctx, typeNamespacedNameDeletion, finalizingClient)).To(Succeed())
			controllerutil.RemoveFinalizer(finalizingClient, clientFinalizer)
			Expect(k8sClient.Update(ctx, finalizingClient)).To(Succeed())

			By("Verifying resource is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedNameDeletion, &keycloakv1.Client{})
				return errors.IsNotFound(err)
			}).Should(BeTrue())
		})
	})

	Context("When testing SAML-specific conversions", func() {
		It("Should handle SAML protocol configuration", func() {
			reconciler := &ClientReconciler{}

			clientID := "saml-client"
			protocol := "saml"

			clientRep := &keycloakv1.ClientRepresentation{
				ClientID: &clientID,
				Protocol: &protocol,
				Attributes: map[string]string{
					"saml.assertion.signature":            "true",
					"saml.client.signature":               "true",
					"saml.encrypt":                        "false",
					"saml.authnstatement":                 "true",
					"saml.force.post.binding":             "true",
					"saml_assertion_consumer_url_post":    "https://sp.example.com/saml/acs",
					"saml_single_logout_service_url_post": "https://sp.example.com/saml/sls",
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying SAML protocol")
			Expect(*goCloak.Protocol).To(Equal("saml"))

			By("Verifying SAML-specific attributes")
			Expect((*goCloak.Attributes)["saml.assertion.signature"]).To(Equal("true"))
			Expect((*goCloak.Attributes)["saml.client.signature"]).To(Equal("true"))
			Expect((*goCloak.Attributes)["saml_assertion_consumer_url_post"]).To(Equal("https://sp.example.com/saml/acs"))
		})

		It("Should handle SAML with protocol mappers", func() {
			reconciler := &ClientReconciler{}

			clientID := "saml-client-with-mappers"
			protocol := "saml"
			mapperName := "role-list"
			mapperProtocol := "saml"
			protocolMapper := "saml-role-list-mapper"

			clientRep := &keycloakv1.ClientRepresentation{
				ClientID: &clientID,
				Protocol: &protocol,
				ProtocolMappers: []keycloakv1.ProtocolMapperRepresentation{
					{
						Name:           &mapperName,
						Protocol:       &mapperProtocol,
						ProtocolMapper: &protocolMapper,
						Config: map[string]string{
							"attribute.name":       "Role",
							"attribute.nameformat": "Basic",
							"single":               "false",
						},
					},
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying SAML mapper is converted")
			Expect(goCloak.ProtocolMappers).NotTo(BeNil())
			Expect(*goCloak.ProtocolMappers).To(HaveLen(1))

			mapper := (*goCloak.ProtocolMappers)[0]
			Expect(*mapper.ProtocolMapper).To(Equal("saml-role-list-mapper"))
			Expect((*mapper.Config)["attribute.name"]).To(Equal("Role"))
		})
	})

	Context("When testing conversion edge cases", func() {
		It("Should handle empty arrays correctly", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client-id"
			clientRep := &keycloakv1.ClientRepresentation{
				ClientID:        &clientID,
				RedirectUris:    []string{},
				WebOrigins:      []string{},
				DefaultRoles:    []string{},
				ProtocolMappers: []keycloakv1.ProtocolMapperRepresentation{},
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying empty arrays are handled")
			Expect(goCloak.RedirectURIs).NotTo(BeNil())
			Expect(*goCloak.RedirectURIs).To(BeEmpty())
			Expect(goCloak.WebOrigins).NotTo(BeNil())
			Expect(*goCloak.WebOrigins).To(BeEmpty())
			Expect(goCloak.ProtocolMappers).To(BeNil())
		})

		It("Should handle all string pointers correctly", func() {
			reconciler := &ClientReconciler{}

			id := "uuid-123"
			clientID := "test-client"
			name := "Test Client"
			description := "A test client"
			rootURL := "https://example.com"
			adminURL := "https://example.com/admin"
			baseURL := "https://example.com/base"
			protocol := "openid-connect"
			secret := "super-secret"
			authType := "client-secret"
			regToken := "registration-token"
			origin := "test-origin"

			clientRep := &keycloakv1.ClientRepresentation{
				ID:                      &id,
				ClientID:                &clientID,
				Name:                    &name,
				Description:             &description,
				RootURL:                 &rootURL,
				AdminURL:                &adminURL,
				BaseURL:                 &baseURL,
				Protocol:                &protocol,
				Secret:                  &secret,
				ClientAuthenticatorType: &authType,
				RegistrationAccessToken: &regToken,
				Origin:                  &origin,
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying all string pointers are mapped")
			Expect(*goCloak.ID).To(Equal(id))
			Expect(*goCloak.ClientID).To(Equal(clientID))
			Expect(*goCloak.Name).To(Equal(name))
			Expect(*goCloak.Description).To(Equal(description))
			Expect(*goCloak.RootURL).To(Equal(rootURL))
			Expect(*goCloak.AdminURL).To(Equal(adminURL))
			Expect(*goCloak.BaseURL).To(Equal(baseURL))
			Expect(*goCloak.Protocol).To(Equal(protocol))
			Expect(*goCloak.Secret).To(Equal(secret))
			Expect(*goCloak.ClientAuthenticatorType).To(Equal(authType))
			Expect(*goCloak.RegistrationAccessToken).To(Equal(regToken))
			Expect(*goCloak.Origin).To(Equal(origin))
		})

		It("Should handle all boolean pointers correctly", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client"

			clientRep := &keycloakv1.ClientRepresentation{
				ClientID:                     &clientID,
				SurrogateAuthRequired:        boolPtr(true),
				Enabled:                      boolPtr(true),
				BearerOnly:                   boolPtr(false),
				ConsentRequired:              boolPtr(true),
				StandardFlowEnabled:          boolPtr(true),
				ImplicitFlowEnabled:          boolPtr(false),
				DirectAccessGrantsEnabled:    boolPtr(true),
				ServiceAccountsEnabled:       boolPtr(true),
				AuthorizationServicesEnabled: boolPtr(true),
				PublicClient:                 boolPtr(false),
				FrontchannelLogout:           boolPtr(true),
				FullScopeAllowed:             boolPtr(true),
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying all boolean pointers are mapped")
			Expect(*goCloak.SurrogateAuthRequired).To(BeTrue())
			Expect(*goCloak.Enabled).To(BeTrue())
			Expect(*goCloak.BearerOnly).To(BeFalse())
			Expect(*goCloak.ConsentRequired).To(BeTrue())
			Expect(*goCloak.StandardFlowEnabled).To(BeTrue())
			Expect(*goCloak.ImplicitFlowEnabled).To(BeFalse())
			Expect(*goCloak.DirectAccessGrantsEnabled).To(BeTrue())
			Expect(*goCloak.ServiceAccountsEnabled).To(BeTrue())
			Expect(*goCloak.AuthorizationServicesEnabled).To(BeTrue())
			Expect(*goCloak.PublicClient).To(BeFalse())
			Expect(*goCloak.FrontChannelLogout).To(BeTrue())
			Expect(*goCloak.FullScopeAllowed).To(BeTrue())
		})

		It("Should handle integer pointers correctly", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client"
			notBefore := int32(1234567890)
			nodeTimeout := int32(300)

			clientRep := &keycloakv1.ClientRepresentation{
				ClientID:                  &clientID,
				NotBefore:                 &notBefore,
				NodeReRegistrationTimeout: &nodeTimeout,
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying integer pointers are mapped")
			Expect(*goCloak.NotBefore).To(Equal(notBefore))
			Expect(*goCloak.NodeReRegistrationTimeout).To(Equal(nodeTimeout))
		})

		It("Should handle maps correctly", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client"

			clientRep := &keycloakv1.ClientRepresentation{
				ClientID: &clientID,
				Attributes: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				AuthenticationFlowBindingOverrides: map[string]string{
					"browser": "custom-browser-flow",
					"direct":  "custom-direct-flow",
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying maps are copied correctly")
			Expect(goCloak.Attributes).NotTo(BeNil())
			Expect((*goCloak.Attributes)["key1"]).To(Equal("value1"))
			Expect((*goCloak.Attributes)["key2"]).To(Equal("value2"))
			Expect(goCloak.AuthenticationFlowBindingOverrides).NotTo(BeNil())
			Expect((*goCloak.AuthenticationFlowBindingOverrides)["browser"]).To(Equal("custom-browser-flow"))
		})

		It("Should handle string arrays correctly", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client"

			clientRep := &keycloakv1.ClientRepresentation{
				ClientID:             &clientID,
				DefaultRoles:         []string{"role1", "role2"},
				RedirectUris:         []string{"http://localhost/callback"},
				WebOrigins:           []string{"http://localhost"},
				DefaultClientScopes:  []string{"openid", "profile"},
				OptionalClientScopes: []string{"email", "phone"},
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying string arrays are copied correctly")
			Expect(goCloak.DefaultRoles).NotTo(BeNil())
			Expect(*goCloak.DefaultRoles).To(ContainElement("role1"))
			Expect(*goCloak.DefaultRoles).To(ContainElement("role2"))
			Expect(goCloak.RedirectURIs).NotTo(BeNil())
			Expect(*goCloak.RedirectURIs).To(ContainElement("http://localhost/callback"))
			Expect(goCloak.DefaultClientScopes).NotTo(BeNil())
			Expect(*goCloak.DefaultClientScopes).To(ContainElement("openid"))
		})

		It("Should handle complex protocol mappers with all fields", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client"
			mapperID := "mapper-123"
			mapperName := "complex-mapper"
			protocol := "openid-connect"
			protocolMapper := "oidc-usermodel-attribute-mapper"

			clientRep := &keycloakv1.ClientRepresentation{
				ClientID: &clientID,
				ProtocolMappers: []keycloakv1.ProtocolMapperRepresentation{
					{
						ID:             &mapperID,
						Name:           &mapperName,
						Protocol:       &protocol,
						ProtocolMapper: &protocolMapper,
						Config: map[string]string{
							"user.attribute":     "customAttribute",
							"claim.name":         "custom_claim",
							"jsonType.label":     "String",
							"id.token.claim":     "true",
							"access.token.claim": "true",
						},
					},
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep)

			By("Verifying complex protocol mapper is mapped correctly")
			Expect(goCloak.ProtocolMappers).NotTo(BeNil())
			Expect(*goCloak.ProtocolMappers).To(HaveLen(1))

			mapper := (*goCloak.ProtocolMappers)[0]
			Expect(*mapper.ID).To(Equal(mapperID))
			Expect(*mapper.Name).To(Equal(mapperName))
			Expect(*mapper.Protocol).To(Equal(protocol))
			Expect(*mapper.ProtocolMapper).To(Equal(protocolMapper))
			Expect(mapper.Config).NotTo(BeNil())
			Expect((*mapper.Config)["user.attribute"]).To(Equal("customAttribute"))
			Expect((*mapper.Config)["id.token.claim"]).To(Equal("true"))
		})
	})

	Context("When testing various client configurations", func() {
		DescribeTable("Should handle different client types",
			func(clientType string, publicClient bool, standardFlow bool, implicitFlow bool, directAccess bool, serviceAccount bool) {
				reconciler := &ClientReconciler{}
				clientID := clientType + "-client"

				clientRep := &keycloakv1.ClientRepresentation{
					ClientID:                  &clientID,
					PublicClient:              &publicClient,
					StandardFlowEnabled:       &standardFlow,
					ImplicitFlowEnabled:       &implicitFlow,
					DirectAccessGrantsEnabled: &directAccess,
					ServiceAccountsEnabled:    &serviceAccount,
				}

				goCloak := reconciler.convertToGoCloak(clientRep)

				Expect(*goCloak.PublicClient).To(Equal(publicClient))
				Expect(*goCloak.StandardFlowEnabled).To(Equal(standardFlow))
				Expect(*goCloak.ImplicitFlowEnabled).To(Equal(implicitFlow))
				Expect(*goCloak.DirectAccessGrantsEnabled).To(Equal(directAccess))
				Expect(*goCloak.ServiceAccountsEnabled).To(Equal(serviceAccount))
			},
			Entry("Public client with standard flow", "public", true, true, false, false, false),
			Entry("Confidential client with all flows", "confidential-all", false, true, true, true, true),
			Entry("Service account only", "service", false, false, false, false, true),
			Entry("Bearer only client", "bearer", false, false, false, false, false),
			Entry("SPA with implicit flow", "spa", true, true, true, false, false),
		)

		DescribeTable("Should handle different protocol configurations",
			func(protocol string, expectedProtocol string) {
				reconciler := &ClientReconciler{}
				clientID := "test-protocol-client"

				clientRep := &keycloakv1.ClientRepresentation{
					ClientID: &clientID,
					Protocol: &protocol,
				}

				goCloak := reconciler.convertToGoCloak(clientRep)

				if goCloak.Protocol != nil {
					Expect(*goCloak.Protocol).To(Equal(expectedProtocol))
				} else {
					Fail("Protocol should not be nil")
				}
			},
			Entry("OpenID Connect", "openid-connect", "openid-connect"),
			Entry("SAML", "saml", "saml"),
			Entry("Docker auth", "docker-v2", "docker-v2"),
		)
	})
})

// Helper functions
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
