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

package utils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	gocloak "github.com/Nerzal/gocloak/v13"
	. "github.com/onsi/ginkgo/v2" // nolint:revive,staticcheck
)

const (
	certmanagerVersion = "v1.19.1"
	certmanagerURLTmpl = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"

	defaultKindBinary  = "kind"
	defaultKindCluster = "kind"

	keycloakNamespace      = "keycloak"
	keycloakHelmRelease    = "keycloak"
	keycloakAdminUser      = "admin"
	keycloakAdminPass      = "admin"
	keycloakLocalPort      = "18080"
	operatorClientID       = "keycloak-operator"
	operatorClientRealm    = "master"
	keycloakHelmChart = "oci://ghcr.io/cloudpirates-io/helm-charts/keycloak"
	// KeycloakCredSecret is the k8s secret name used to pass Keycloak credentials to the operator.
	KeycloakCredSecret = "keycloak-operator-credentials"
)

// KeycloakCredentials holds the credentials for the operator to authenticate with Keycloak.
type KeycloakCredentials struct {
	URL      string
	ClientID string
	Secret   string
	Realm    string
}

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// InstallKeycloak deploys Keycloak to the kind cluster using the CloudPirates helm chart.
// The version parameter sets the Keycloak image tag (e.g. "26.2.4").
func InstallKeycloak(version string) error {
	_, _ = fmt.Fprintf(GinkgoWriter, "Installing Keycloak %s...\n", version)

	// Create namespace (ignore error if it already exists)
	// #nosec G204 -- test utility with controlled options
	cmd := exec.Command("kubectl", "create", "ns", keycloakNamespace)
	_, _ = Run(cmd)

	// #nosec G204 -- test utility with controlled options
	cmd = exec.Command("helm", "upgrade", "--install", keycloakHelmRelease,
		keycloakHelmChart,
		"--namespace", keycloakNamespace,
		"--set", "production=false",
		"--set", "auth.adminUser="+keycloakAdminUser,
		"--set", "auth.adminPassword="+keycloakAdminPass,
		"--set", "postgresql.enabled=false",
		"--set", "image.tag="+version,
		"--wait",
		"--timeout", "10m",
	)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to install Keycloak: %w", err)
	}
	return nil
}

// SetupKeycloakOperatorAccess configures a Keycloak service account client for the operator
// and returns its credentials. It uses a temporary kubectl port-forward to reach Keycloak.
func SetupKeycloakOperatorAccess() (*KeycloakCredentials, error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "Setting up Keycloak operator service account...\n")

	// Discover service name
	// #nosec G204 -- test utility
	cmd := exec.Command("kubectl", "get", "svc", "-n", keycloakNamespace,
		"-o", "jsonpath={.items[0].metadata.name}")
	svcName, err := Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get Keycloak service: %w", err)
	}
	svcName = strings.TrimSpace(svcName)
	if svcName == "" {
		return nil, fmt.Errorf("no service found in namespace %s", keycloakNamespace)
	}

	// Start port-forward
	// #nosec G204 -- test utility with controlled options
	pfCmd := exec.Command("kubectl", "port-forward",
		fmt.Sprintf("svc/%s", svcName),
		"-n", keycloakNamespace,
		keycloakLocalPort+":8080",
	)
	if err := pfCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start port-forward: %w", err)
	}
	defer func() { _ = pfCmd.Process.Kill() }()

	if err := waitForPort(keycloakLocalPort, 30*time.Second); err != nil {
		return nil, fmt.Errorf("Keycloak port-forward not ready: %w", err)
	}

	// Configure via gocloak
	gc := gocloak.NewClient("http://localhost:" + keycloakLocalPort)
	gctx := context.Background()

	token, err := gc.Login(gctx, "admin-cli", "", operatorClientRealm, keycloakAdminUser, keycloakAdminPass)
	if err != nil {
		return nil, fmt.Errorf("failed to login to Keycloak: %w", err)
	}

	// Remove any existing operator client
	existing, _ := gc.GetClients(gctx, token.AccessToken, operatorClientRealm,
		gocloak.GetClientsParams{ClientID: gocloak.StringP(operatorClientID)})
	for _, c := range existing {
		_ = gc.DeleteClient(gctx, token.AccessToken, operatorClientRealm, *c.ID)
	}

	// Create service account client
	svcEnabled := true
	newClient := gocloak.Client{
		ClientID:               gocloak.StringP(operatorClientID),
		ServiceAccountsEnabled: &svcEnabled,
		ClientAuthenticatorType: gocloak.StringP("client-secret"),
	}
	clientInternalID, err := gc.CreateClient(gctx, token.AccessToken, operatorClientRealm, newClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create operator client: %w", err)
	}

	// Assign realm-admin role to the service account
	saUser, err := gc.GetClientServiceAccount(gctx, token.AccessToken, operatorClientRealm, clientInternalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get service account user: %w", err)
	}

	realmMgmtClients, err := gc.GetClients(gctx, token.AccessToken, operatorClientRealm,
		gocloak.GetClientsParams{ClientID: gocloak.StringP("realm-management")})
	if err != nil || len(realmMgmtClients) == 0 {
		return nil, fmt.Errorf("failed to get realm-management client: %w", err)
	}
	realmMgmtID := *realmMgmtClients[0].ID

	adminRole, err := gc.GetClientRole(gctx, token.AccessToken, operatorClientRealm, realmMgmtID, "realm-admin")
	if err != nil {
		return nil, fmt.Errorf("failed to get realm-admin role: %w", err)
	}

	if err := gc.AddClientRolesToUser(gctx, token.AccessToken, operatorClientRealm,
		realmMgmtID, *saUser.ID, []gocloak.Role{*adminRole}); err != nil {
		return nil, fmt.Errorf("failed to assign realm-admin role: %w", err)
	}

	// Retrieve generated client secret
	creds, err := gc.GetClientSecret(gctx, token.AccessToken, operatorClientRealm, clientInternalID)
	if err != nil || creds.Value == nil {
		return nil, fmt.Errorf("failed to get client secret: %w", err)
	}

	keycloakURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:8080", svcName, keycloakNamespace)
	return &KeycloakCredentials{
		URL:      keycloakURL,
		ClientID: operatorClientID,
		Secret:   *creds.Value,
		Realm:    operatorClientRealm,
	}, nil
}

// UninstallKeycloak removes Keycloak from the cluster.
func UninstallKeycloak() {
	_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling Keycloak...\n")
	// #nosec G204 -- test utility
	cmd := exec.Command("helm", "uninstall", keycloakHelmRelease,
		"--namespace", keycloakNamespace, "--ignore-not-found")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
	// #nosec G204 -- test utility
	cmd = exec.Command("kubectl", "delete", "ns", keycloakNamespace, "--ignore-not-found")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// waitForPort blocks until the given TCP port on localhost is accepting connections or timeout elapses.
func waitForPort(port string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "localhost:"+port, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("port %s not ready after %v", port, timeout)
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) (string, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %q\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %q\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%q failed with error %q: %w", command, string(output), err)
	}

	return string(output), nil
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	// #nosec G204 -- test utility with safe URL
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}

	// Delete leftover leases in kube-system (not cleaned by default)
	kubeSystemLeases := []string{
		"cert-manager-cainjector-leader-election",
		"cert-manager-controller",
	}
	for _, lease := range kubeSystemLeases {
		// #nosec G204 -- test utility with predefined lease names
		cmd = exec.Command("kubectl", "delete", "lease", lease,
			"-n", "kube-system", "--ignore-not-found", "--force", "--grace-period=0")
		if _, err := Run(cmd); err != nil {
			warnError(err)
		}
	}
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	// #nosec G204 -- test utility with safe URL
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

// IsCertManagerCRDsInstalled checks if any Cert Manager CRDs are installed
// by verifying the existence of key CRDs related to Cert Manager.
func IsCertManagerCRDsInstalled() bool {
	// List of common Cert Manager CRDs
	certManagerCRDs := []string{
		"certificates.cert-manager.io",
		"issuers.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"certificaterequests.cert-manager.io",
		"orders.acme.cert-manager.io",
		"challenges.acme.cert-manager.io",
	}

	// Execute the kubectl command to get all CRDs
	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	// Check if any of the Cert Manager CRDs are present
	crdList := GetNonEmptyLines(output)
	for _, crd := range certManagerCRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

// LoadImageToKindClusterWithName loads a local docker image to the kind cluster
func LoadImageToKindClusterWithName(name string) error {
	cluster := defaultKindCluster
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	kindBinary := defaultKindBinary
	if v, ok := os.LookupEnv("KIND"); ok {
		kindBinary = v
	}
	// #nosec G204 -- test utility with controlled options
	cmd := exec.Command(kindBinary, kindOptions...)
	_, err := Run(cmd)
	return err
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, fmt.Errorf("failed to get current working directory: %w", err)
	}
	wd = strings.ReplaceAll(wd, "/test/e2e", "")
	return wd, nil
}

// UncommentCode searches for target in the file and remove the comment prefix
// of the target content. The target content may span multiple lines.
func UncommentCode(filename, target, prefix string) error {
	// #nosec G304 -- test utility with caller-provided path
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}
	strContent := string(content)

	idx := strings.Index(strContent, target)
	if idx < 0 {
		return fmt.Errorf("unable to find the code %q to be uncomment", target)
	}

	out := new(bytes.Buffer)
	// nolint:gosec // G304: test utility reading files with caller-provided paths
	_, err = out.Write(content[:idx])
	if err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewBufferString(target))
	if !scanner.Scan() {
		return nil
	}
	for {
		if _, err = out.WriteString(strings.TrimPrefix(scanner.Text(), prefix)); err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
		// Avoid writing a newline in case the previous line was the last in target.
		if !scanner.Scan() {
			break
		}
		if _, err = out.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
	}

	if _, err = out.Write(content[idx+len(target):]); err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}

	// #nosec G306 -- test utility needs readable permissions
	if err = os.WriteFile(filename, out.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", filename, err)
	}

	return nil
}
