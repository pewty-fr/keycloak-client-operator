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
	"os"

	gocloak "github.com/Nerzal/gocloak/v13"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	keycloakv1 "github.com/pewty-fr/keycloak-client-operator/api/v1"
)

const (
	testClientID = "test-client-id"
	protocolOIDC = "openid-connect"
	protocolSAML = "saml"
	realmMaster  = "master"
	testRealm    = "test-realm"
)

var _ = Describe("Client Controller", func() {
	BeforeEach(func() {
		// Set environment variables for Keycloak connection
		Expect(os.Setenv("KEYCLOAK_URL", "http://localhost:8080")).To(Succeed())
		Expect(os.Setenv("KEYCLOAK_USER", "admin")).To(Succeed())
		Expect(os.Setenv("KEYCLOAK_PASSWORD", "admin")).To(Succeed())
	})

	Context("When converting ClientRepresentation to gocloak.Client", func() {
		It("Should correctly map all basic fields", func() {
			reconciler := &ClientReconciler{}

			clientID := testClientID
			clientName := "Test Client"
			clientSecret := "test-secret"
			enabled := true
			publicClient := false

			clientRep := &keycloakv1.ClientRepresentation{
				Name:                      &clientName,
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

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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

			clientID := testClientID
			clientSecret := ""
			mapperName := "email"
			protocol := protocolOIDC
			protocolMapper := "oidc-usermodel-property-mapper"

			clientRep := &keycloakv1.ClientRepresentation{
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

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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

			clientID := testClientID
			clientSecret := ""
			clientRep := &keycloakv1.ClientRepresentation{
				RegisteredNodes: map[string]int32{
					"node1": 12345,
					"node2": 67890,
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

			By("Verifying RegisteredNodes conversion from int32 to int")
			Expect(goCloak.RegisteredNodes).NotTo(BeNil())
			Expect((*goCloak.RegisteredNodes)["node1"]).To(Equal(12345))
			Expect((*goCloak.RegisteredNodes)["node2"]).To(Equal(67890))
		})

		It("Should handle Access map type conversion", func() {
			reconciler := &ClientReconciler{}

			clientID := testClientID
			clientSecret := ""
			clientRep := &keycloakv1.ClientRepresentation{
				Access: map[string]bool{
					"view":      true,
					"configure": false,
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

			By("Verifying Access conversion from bool to interface{}")
			Expect(goCloak.Access).NotTo(BeNil())
			Expect((*goCloak.Access)["view"]).To(BeTrue())
			Expect((*goCloak.Access)["configure"]).To(BeFalse())
		})

		It("Should handle nil values gracefully", func() {
			reconciler := &ClientReconciler{}

			clientID := testClientID
			clientSecret := ""
			clientRep := &keycloakv1.ClientRepresentation{}

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("Should create a valid Client resource with required fields", func() {
			By("Creating the custom resource with required fields")

			realm := realmMaster
			secretRef := keycloakv1.ClientSecretReference{Name: "test-client-credentials"}

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: secretRef,
					Client: keycloakv1.ClientRepresentation{
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
			Expect(createdClient.Spec.SecretRef.Name).To(Equal("test-client-credentials"))

			By("Verifying finalizer is added by the controller")
			// Note: Finalizer addition requires the controller to reconcile,
			// which requires a running Keycloak instance in integration tests
			// This is tested in e2e tests instead
		})

		It("Should reject a Client resource without required realm", func() {
			By("Attempting to create resource without realm")

			secretRef := keycloakv1.ClientSecretReference{Name: "test-client-credentials"}

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-client",
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm:     nil, // Missing required field
					SecretRef: secretRef,
					Client:    keycloakv1.ClientRepresentation{},
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
			Expect(result.RequeueAfter).To(BeZero())
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
					Expect(k8sClient.Update(ctx, resource)).To(Succeed())
				}
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("Should return error when realm is nil during reconciliation", func() {
			By("Creating a resource that somehow has nil realm (bypassing CRD validation)")
			// This tests the controller's validation logic
			reconciler := &ClientReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Create resource with realm (passes CRD validation)
			realm := realmMaster
			secretRef := keycloakv1.ClientSecretReference{Name: "validation-secret"}
			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: secretRef,
					Client:    keycloakv1.ClientRepresentation{},
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

		It("Should return error when secretRef.name is empty during reconciliation", func() {
			reconciler := &ClientReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			realm := realmMaster
			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName + "-2",
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: keycloakv1.ClientSecretReference{Name: ""},
					Client:    keycloakv1.ClientRepresentation{},
				},
			}
			err := k8sClient.Create(ctx, resource)

			By("Attempting reconciliation should fail validation")
			if err == nil {
				typeNamespacedName2 := types.NamespacedName{
					Name:      resourceName + "-2",
					Namespace: "default",
				}
				req := ctrl.Request{NamespacedName: typeNamespacedName2}
				_, reconcileErr := reconciler.Reconcile(ctx, req)
				Expect(reconcileErr).To(HaveOccurred())
				Expect(reconcileErr.Error()).To(ContainSubstring("secretRef.name is required"))
			} else {
				// CRD validation prevented the creation
				Expect(err).To(HaveOccurred())
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
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("Should update status conditions", func() {
			By("Creating a Client resource")
			realm := testRealm
			secretRef := keycloakv1.ClientSecretReference{Name: "status-secret"}

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: secretRef,
					Client:    keycloakv1.ClientRepresentation{},
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
			realm := testRealm
			secretRef := keycloakv1.ClientSecretReference{Name: "status-secret-update"}

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName + "-update",
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: secretRef,
					Client:    keycloakv1.ClientRepresentation{},
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
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
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
					Expect(k8sClient.Update(ctx, resource)).To(Succeed())
				}
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("Should add finalizer to new resource", func() {
			By("Creating a Client resource")
			realm := testRealm
			secretRef := keycloakv1.ClientSecretReference{Name: "finalizer-secret"}

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: secretRef,
					Client:    keycloakv1.ClientRepresentation{},
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
			realm := testRealm
			secretRef := keycloakv1.ClientSecretReference{Name: "finalizer-secret-del"}

			resource := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName + "-deletion",
					Namespace:  "default",
					Finalizers: []string{clientFinalizer},
				},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: secretRef,
					Client:    keycloakv1.ClientRepresentation{},
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
			clientSecret := ""
			protocol := protocolSAML

			clientRep := &keycloakv1.ClientRepresentation{
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

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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
			clientSecret := ""
			protocol := protocolSAML
			mapperName := "role-list"
			mapperProtocol := "saml"
			protocolMapper := "saml-role-list-mapper"

			clientRep := &keycloakv1.ClientRepresentation{
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

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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

			clientID := testClientID
			clientSecret := ""
			clientRep := &keycloakv1.ClientRepresentation{
				RedirectUris:    []string{},
				WebOrigins:      []string{},
				DefaultRoles:    []string{},
				ProtocolMappers: []keycloakv1.ProtocolMapperRepresentation{},
			}

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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
			clientID := "test-client" //nolint:goconst // Local variable in test context
			clientSecret := "super-secret"
			name := "Test Client"
			description := "A test client"
			rootURL := "https://example.com"
			adminURL := "https://example.com/admin"
			baseURL := "https://example.com/base"
			protocol := protocolOIDC
			authType := "client-secret"
			regToken := "registration-token"
			origin := "test-origin"

			clientRep := &keycloakv1.ClientRepresentation{
				ID:                      &id,
				Name:                    &name,
				Description:             &description,
				RootURL:                 &rootURL,
				AdminURL:                &adminURL,
				BaseURL:                 &baseURL,
				Protocol:                &protocol,
				ClientAuthenticatorType: &authType,
				RegistrationAccessToken: &regToken,
				Origin:                  &origin,
			}

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

			By("Verifying all string pointers are mapped")
			Expect(*goCloak.ID).To(Equal(id))
			Expect(*goCloak.ClientID).To(Equal(clientID))
			Expect(*goCloak.Name).To(Equal(name))
			Expect(*goCloak.Description).To(Equal(description))
			Expect(*goCloak.RootURL).To(Equal(rootURL))
			Expect(*goCloak.AdminURL).To(Equal(adminURL))
			Expect(*goCloak.BaseURL).To(Equal(baseURL))
			Expect(*goCloak.Protocol).To(Equal(protocol))
			Expect(*goCloak.Secret).To(Equal(clientSecret))
			Expect(*goCloak.ClientAuthenticatorType).To(Equal(authType))
			Expect(*goCloak.RegistrationAccessToken).To(Equal(regToken))
			Expect(*goCloak.Origin).To(Equal(origin))
		})

		It("Should handle all boolean pointers correctly", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client"
			clientSecret := ""

			clientRep := &keycloakv1.ClientRepresentation{
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

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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
			clientSecret := ""
			notBefore := int32(1234567890)
			nodeTimeout := int32(300)

			clientRep := &keycloakv1.ClientRepresentation{
				NotBefore:                 &notBefore,
				NodeReRegistrationTimeout: &nodeTimeout,
			}

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

			By("Verifying integer pointers are mapped")
			Expect(*goCloak.NotBefore).To(Equal(notBefore))
			Expect(*goCloak.NodeReRegistrationTimeout).To(Equal(nodeTimeout))
		})

		It("Should handle maps correctly", func() {
			reconciler := &ClientReconciler{}

			clientID := "test-client"
			clientSecret := ""

			clientRep := &keycloakv1.ClientRepresentation{
				Attributes: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				AuthenticationFlowBindingOverrides: map[string]string{
					"browser": "custom-browser-flow",
					"direct":  "custom-direct-flow",
				},
			}

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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
			clientSecret := ""

			clientRep := &keycloakv1.ClientRepresentation{
				DefaultRoles:         []string{"role1", "role2"},
				RedirectUris:         []string{"http://localhost/callback"},
				WebOrigins:           []string{"http://localhost"},
				DefaultClientScopes:  []string{"openid", "profile"},
				OptionalClientScopes: []string{"email", "phone"},
			}

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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
			clientSecret := ""
			mapperID := "mapper-123"
			mapperName := "complex-mapper"
			protocol := protocolOIDC
			protocolMapper := "oidc-usermodel-attribute-mapper"

			clientRep := &keycloakv1.ClientRepresentation{
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

			goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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
				clientSecret := ""

				clientRep := &keycloakv1.ClientRepresentation{
					PublicClient:              &publicClient,
					StandardFlowEnabled:       &standardFlow,
					ImplicitFlowEnabled:       &implicitFlow,
					DirectAccessGrantsEnabled: &directAccess,
					ServiceAccountsEnabled:    &serviceAccount,
				}

				goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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
				clientSecret := ""

				clientRep := &keycloakv1.ClientRepresentation{
					Protocol: &protocol,
				}

				goCloak := reconciler.convertToGoCloak(clientRep, clientID, clientSecret)

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

	Context("When testing getClientCredentials", func() {
		const secretNamespace = "default"

		AfterEach(func() {
			_ = k8sClient.DeleteAllOf(ctx, &corev1.Secret{}, client.InNamespace(secretNamespace),
				client.MatchingLabels{"test-group": "get-credentials"})
		})

		It("Should read clientId and clientSecret from secret", func() {
			secretName := "creds-test-secret"
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: secretName, Namespace: secretNamespace,
					Labels: map[string]string{"test-group": "get-credentials"},
				},
				Data: map[string][]byte{
					"clientId":     []byte("my-client"),
					"clientSecret": []byte("my-secret"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			realm := realmMaster
			kcClient := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{Name: "creds-test", Namespace: secretNamespace},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: keycloakv1.ClientSecretReference{Name: secretName},
				},
			}
			reconciler := &ClientReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			id, sec, err := reconciler.getClientCredentials(ctx, kcClient)

			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("my-client"))
			Expect(sec).To(Equal("my-secret"))
		})

		It("Should use custom key names from SecretRef", func() {
			secretName := "creds-custom-secret"
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: secretName, Namespace: secretNamespace,
					Labels: map[string]string{"test-group": "get-credentials"},
				},
				Data: map[string][]byte{
					"custom.id":  []byte("custom-client"),
					"custom.sec": []byte("custom-secret"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			realm := realmMaster
			kcClient := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{Name: "creds-custom-test", Namespace: secretNamespace},
				Spec: keycloakv1.ClientSpec{
					Realm: &realm,
					SecretRef: keycloakv1.ClientSecretReference{
						Name:            secretName,
						ClientIDKey:     "custom.id",
						ClientSecretKey: "custom.sec",
					},
				},
			}
			reconciler := &ClientReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			id, sec, err := reconciler.getClientCredentials(ctx, kcClient)

			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("custom-client"))
			Expect(sec).To(Equal("custom-secret"))
		})

		It("Should return error when secret does not exist", func() {
			realm := realmMaster
			kcClient := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{Name: "creds-missing-test", Namespace: secretNamespace},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: keycloakv1.ClientSecretReference{Name: "nonexistent-secret"},
				},
			}
			reconciler := &ClientReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			_, _, err := reconciler.getClientCredentials(ctx, kcClient)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nonexistent-secret"))
		})

		It("Should return empty secret when clientSecret key is missing", func() {
			secretName := "creds-no-secret-key"
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: secretName, Namespace: secretNamespace,
					Labels: map[string]string{"test-group": "get-credentials"},
				},
				Data: map[string][]byte{
					"clientId": []byte("only-id"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			realm := realmMaster
			kcClient := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{Name: "creds-no-secret-test", Namespace: secretNamespace},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: keycloakv1.ClientSecretReference{Name: secretName},
				},
			}
			reconciler := &ClientReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			id, sec, err := reconciler.getClientCredentials(ctx, kcClient)

			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("only-id"))
			Expect(sec).To(BeEmpty())
		})
	})

	Context("When testing updateSecretWithCredentials", func() {
		It("Should update existing secret with new credentials", func() {
			secretName := "update-creds-secret"
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "default"},
				Data:       map[string][]byte{"clientId": []byte("old-id"), "clientSecret": []byte("old-secret")},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() { _ = k8sClient.Delete(ctx, secret) }()

			realm := realmMaster
			kcClient := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{Name: "update-creds-test", Namespace: "default"},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: keycloakv1.ClientSecretReference{Name: secretName},
				},
			}
			reconciler := &ClientReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			newID := "new-client-id"
			newSecret := "new-client-secret"
			err := reconciler.updateSecretWithCredentials(ctx, kcClient, &newID, &newSecret)
			Expect(err).NotTo(HaveOccurred())

			updated := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, updated)).To(Succeed())
			Expect(string(updated.Data["clientId"])).To(Equal("new-client-id"))
			Expect(string(updated.Data["clientSecret"])).To(Equal("new-client-secret"))
		})

		It("Should skip update when credentials are nil", func() {
			secretName := "nil-creds-secret"
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "default"},
				Data:       map[string][]byte{"clientId": []byte("original")},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() { _ = k8sClient.Delete(ctx, secret) }()

			realm := realmMaster
			kcClient := &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{Name: "nil-creds-test", Namespace: "default"},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: keycloakv1.ClientSecretReference{Name: secretName},
				},
			}
			reconciler := &ClientReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			err := reconciler.updateSecretWithCredentials(ctx, kcClient, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			unchanged := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, unchanged)).To(Succeed())
			Expect(string(unchanged.Data["clientId"])).To(Equal("original"))
		})
	})

	Context("When testing Reconcile with mock Keycloak", func() {
		const reconcileNamespace = "default"

		makeReconcilerWith := func(mock GoCloak) *ClientReconciler {
			return &ClientReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				KeycloakClient: mock,
				KeycloakUser:   "operator-client",
				KeycloakPass:   "operator-secret",
				KeycloakRealm:  realmMaster,
			}
		}

		makeSecret := func(name string) *corev1.Secret {
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: reconcileNamespace},
				Data: map[string][]byte{
					"clientId":     []byte("my-kc-client"),
					"clientSecret": []byte("my-kc-secret"),
				},
			}
		}

		makeClient := func(name, secretName string, finalizers ...string) *keycloakv1.Client {
			realm := realmMaster
			return &keycloakv1.Client{
				ObjectMeta: metav1.ObjectMeta{
					Name:       name,
					Namespace:  reconcileNamespace,
					Finalizers: finalizers,
				},
				Spec: keycloakv1.ClientSpec{
					Realm:     &realm,
					SecretRef: keycloakv1.ClientSecretReference{Name: secretName},
					Client:    keycloakv1.ClientRepresentation{Enabled: boolPtr(true)},
				},
			}
		}

		It("Should return error when Keycloak authentication fails", func() {
			secret := makeSecret("auth-fail-secret")
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() { _ = k8sClient.Delete(ctx, secret) }()

			resource := makeClient("auth-fail-client", "auth-fail-secret", clientFinalizer)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			defer func() {
				controllerutil.RemoveFinalizer(resource, clientFinalizer)
				_ = k8sClient.Update(ctx, resource)
				_ = k8sClient.Delete(ctx, resource)
			}()

			reconciler := makeReconcilerWith(errGoCloak("keycloak unreachable"))
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "auth-fail-client", Namespace: reconcileNamespace}}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("keycloak unreachable"))
		})

		It("Should create a new client in Keycloak when it does not exist", func() {
			secret := makeSecret("create-test-secret")
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() { _ = k8sClient.Delete(ctx, secret) }()

			resource := makeClient("create-test-client", "create-test-secret", clientFinalizer)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			defer func() {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: "create-test-client", Namespace: reconcileNamespace}, resource)
				controllerutil.RemoveFinalizer(resource, clientFinalizer)
				_ = k8sClient.Update(ctx, resource)
				_ = k8sClient.Delete(ctx, resource)
			}()

			createCalled := false
			internalID := "internal-uuid-001"
			retClientID := "my-kc-client"
			retSecret := "keycloak-generated-secret"

			mock := &mockGoCloak{
				GetClientsFunc: func(_ context.Context, _, _ string, _ gocloak.GetClientsParams) ([]*gocloak.Client, error) {
					return []*gocloak.Client{}, nil
				},
				CreateClientFunc: func(_ context.Context, _, _ string, _ gocloak.Client) (string, error) {
					createCalled = true
					return internalID, nil
				},
				GetClientFunc: func(_ context.Context, _, _, id string) (*gocloak.Client, error) {
					Expect(id).To(Equal(internalID))
					return &gocloak.Client{ClientID: &retClientID, Secret: &retSecret}, nil
				},
			}
			reconciler := makeReconcilerWith(mock)
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "create-test-client", Namespace: reconcileNamespace}}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(createCalled).To(BeTrue())

			// Secret should be updated with generated credentials
			updatedSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "create-test-secret", Namespace: reconcileNamespace}, updatedSecret)).To(Succeed())
			Expect(string(updatedSecret.Data["clientSecret"])).To(Equal("keycloak-generated-secret"))
		})

		It("Should update an existing client in Keycloak", func() {
			secret := makeSecret("update-test-secret")
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() { _ = k8sClient.Delete(ctx, secret) }()

			resource := makeClient("update-test-client", "update-test-secret", clientFinalizer)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			defer func() {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: "update-test-client", Namespace: reconcileNamespace}, resource)
				controllerutil.RemoveFinalizer(resource, clientFinalizer)
				_ = k8sClient.Update(ctx, resource)
				_ = k8sClient.Delete(ctx, resource)
			}()

			updateCalled := false
			existingID := "existing-uuid"
			existingClientID := "my-kc-client"

			mock := &mockGoCloak{
				GetClientsFunc: func(_ context.Context, _, _ string, _ gocloak.GetClientsParams) ([]*gocloak.Client, error) {
					return []*gocloak.Client{{ID: &existingID, ClientID: &existingClientID}}, nil
				},
				UpdateClientFunc: func(_ context.Context, _, _ string, c gocloak.Client) error {
					updateCalled = true
					Expect(c.ID).To(Equal(&existingID))
					return nil
				},
			}
			reconciler := makeReconcilerWith(mock)
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "update-test-client", Namespace: reconcileNamespace}}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(updateCalled).To(BeTrue())
		})

		It("Should delete client from Keycloak and remove finalizer on deletion", func() {
			secret := makeSecret("delete-test-secret")
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() { _ = k8sClient.Delete(ctx, secret) }()

			resource := makeClient("delete-test-client", "delete-test-secret", clientFinalizer)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			// Trigger deletion
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			deleteCalled := false
			existingID := "to-delete-uuid"
			existingClientID := "my-kc-client"

			mock := &mockGoCloak{
				GetClientsFunc: func(_ context.Context, _, _ string, _ gocloak.GetClientsParams) ([]*gocloak.Client, error) {
					return []*gocloak.Client{{ID: &existingID, ClientID: &existingClientID}}, nil
				},
				DeleteClientFunc: func(_ context.Context, _, _, id string) error {
					deleteCalled = true
					Expect(id).To(Equal(existingID))
					return nil
				},
			}
			reconciler := makeReconcilerWith(mock)
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "delete-test-client", Namespace: reconcileNamespace}}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteCalled).To(BeTrue())

			// Resource should have finalizer removed and be deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "delete-test-client", Namespace: reconcileNamespace}, &keycloakv1.Client{})
				return errors.IsNotFound(err)
			}).Should(BeTrue())
		})

		It("Should add finalizer on first reconciliation and requeue", func() {
			secret := makeSecret("finalizer-add-secret")
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() { _ = k8sClient.Delete(ctx, secret) }()

			// Create without finalizer
			resource := makeClient("finalizer-add-client", "finalizer-add-secret")
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			defer func() {
				r := &keycloakv1.Client{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: "finalizer-add-client", Namespace: reconcileNamespace}, r); err == nil {
					controllerutil.RemoveFinalizer(r, clientFinalizer)
					_ = k8sClient.Update(ctx, r)
					_ = k8sClient.Delete(ctx, r)
				}
			}()

			reconciler := makeReconcilerWith(&mockGoCloak{})
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "finalizer-add-client", Namespace: reconcileNamespace}}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// Finalizer should be added
			updated := &keycloakv1.Client{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "finalizer-add-client", Namespace: reconcileNamespace}, updated)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(updated, clientFinalizer)).To(BeTrue())
		})

		It("Should return error when CreateClient fails", func() {
			secret := makeSecret("create-fail-secret")
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() { _ = k8sClient.Delete(ctx, secret) }()

			resource := makeClient("create-fail-client", "create-fail-secret", clientFinalizer)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			defer func() {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: "create-fail-client", Namespace: reconcileNamespace}, resource)
				controllerutil.RemoveFinalizer(resource, clientFinalizer)
				_ = k8sClient.Update(ctx, resource)
				_ = k8sClient.Delete(ctx, resource)
			}()

			mock := &mockGoCloak{
				GetClientsFunc: func(_ context.Context, _, _ string, _ gocloak.GetClientsParams) ([]*gocloak.Client, error) {
					return []*gocloak.Client{}, nil
				},
				CreateClientFunc: func(_ context.Context, _, _ string, _ gocloak.Client) (string, error) {
					return "", fmt.Errorf("keycloak create error")
				},
			}
			reconciler := makeReconcilerWith(mock)
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "create-fail-client", Namespace: reconcileNamespace}}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("keycloak create error"))
		})
	})
})

// Helper functions
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
