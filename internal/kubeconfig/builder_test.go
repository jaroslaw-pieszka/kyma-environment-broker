package kubeconfig

import (
	"fmt"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/kyma-environment-broker/internal"

	"github.com/stretchr/testify/require"
)

const (
	globalAccountID = "d9d501c2-bdcb-49f2-8e86-1c4e05b90f5e"
	runtimeID       = "f7d634ae-4ce2-4916-be64-b6fb493155df"

	issuerURL  = "https://example.com"
	issuer2URL = "https://example2.com"
	clientID   = "c1id"
	client2ID  = "c2id"
)

func TestBuilder_BuildFromRuntimeResource_NilAdditionalOIDC(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)

	runtimeResource := &imv1.Runtime{}
	runtimeResource.ObjectMeta.Name = runtimeID
	runtimeResource.ObjectMeta.Namespace = "kcp-system"
	runtimeResource.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = nil

	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(runtimeResource).Build()

	t.Run("new kubeconfig was built properly", func(t *testing.T) {
		// given
		builder := NewBuilder(kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()), true)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		_, err := builder.Build(instance)

		//then
		assert.EqualError(t, err, "while fetching oidc data: Runtime Resource contains no additional OIDC config")
	})
}

func TestBuilder_BuildFromRuntimeResource_EmptyAdditionalOIDC(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)

	runtimeResource := &imv1.Runtime{}
	runtimeResource.ObjectMeta.Name = runtimeID
	runtimeResource.ObjectMeta.Namespace = "kcp-system"
	runtimeResource.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = &[]imv1.OIDCConfig{}

	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(runtimeResource).Build()

	t.Run("new kubeconfig was built properly", func(t *testing.T) {
		// given
		builder := NewBuilder(kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()), true)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.Build(instance)

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newKubeconfigWithNoUsers())
	})
}

func TestBuilder_BuildFromRuntimeResource(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)

	runtimeResource := &imv1.Runtime{}
	runtimeResource.ObjectMeta.Name = runtimeID
	runtimeResource.ObjectMeta.Namespace = "kcp-system"
	runtimeResource.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = &[]imv1.OIDCConfig{
		{
			OIDCConfig: gardener.OIDCConfig{
				ClientID:  ptr.String(clientID),
				IssuerURL: ptr.String(issuerURL),
			},
		},
	}

	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(runtimeResource).Build()

	t.Run("new kubeconfig was built properly", func(t *testing.T) {
		// given
		builder := NewBuilder(kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()), true)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.Build(instance)

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newKubeconfig())
	})
}

func TestBuilder_BuildFromRuntimeResource_MultipleAdditionalOIDC(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)

	runtimeResource := &imv1.Runtime{}
	runtimeResource.ObjectMeta.Name = runtimeID
	runtimeResource.ObjectMeta.Namespace = "kcp-system"
	runtimeResource.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = &[]imv1.OIDCConfig{
		{
			OIDCConfig: gardener.OIDCConfig{
				ClientID:  ptr.String(clientID),
				IssuerURL: ptr.String(issuerURL),
			},
		},
		{
			OIDCConfig: gardener.OIDCConfig{
				ClientID:  ptr.String(client2ID),
				IssuerURL: ptr.String(issuer2URL),
			},
		},
	}

	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(runtimeResource).Build()

	t.Run("new kubeconfig was built properly", func(t *testing.T) {
		// given
		builder := NewBuilder(kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()), true)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.Build(instance)

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newKubeconfigWithMultipleContexts())
	})

	t.Run("should return kubeconfig with one context when feature flag is disabled", func(t *testing.T) {
		// given
		builder := NewBuilder(kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()), false)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.Build(instance)

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newKubeconfig())
	})
}

func TestBuilder_BuildFromAdminKubeconfig_NilAdditionalOIDC(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	runtimeResource := &imv1.Runtime{}
	runtimeResource.ObjectMeta.Name = runtimeID
	runtimeResource.ObjectMeta.Namespace = "kcp-system"
	runtimeResource.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = nil

	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(runtimeResource).Build()

	t.Run("new kubeconfig was build properly", func(t *testing.T) {
		// given

		builder := NewBuilder(kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()), true)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		_, err := builder.BuildFromAdminKubeconfig(instance, adminKubeconfig())

		//then
		require.EqualError(t, err, "while fetching oidc data: Runtime Resource contains no additional OIDC config")
	})
}

func TestBuilder_BuildFromAdminKubeconfig_EmptyAdditionalOIDC(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	runtimeResource := &imv1.Runtime{}
	runtimeResource.ObjectMeta.Name = runtimeID
	runtimeResource.ObjectMeta.Namespace = "kcp-system"
	runtimeResource.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = &[]imv1.OIDCConfig{}

	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(runtimeResource).Build()

	t.Run("new kubeconfig was build properly", func(t *testing.T) {
		// given

		builder := NewBuilder(kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()), true)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.BuildFromAdminKubeconfig(instance, adminKubeconfig())

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newOwnClusterKubeconfigWithNoUsers())
	})
}

func TestBuilder_BuildFromAdminKubeconfig(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	runtimeResource := &imv1.Runtime{}
	runtimeResource.ObjectMeta.Name = runtimeID
	runtimeResource.ObjectMeta.Namespace = "kcp-system"
	runtimeResource.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = &[]imv1.OIDCConfig{
		{
			OIDCConfig: gardener.OIDCConfig{
				ClientID:  ptr.String(clientID),
				IssuerURL: ptr.String(issuerURL),
			},
		},
	}

	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(runtimeResource).Build()

	t.Run("new kubeconfig was build properly", func(t *testing.T) {
		// given

		builder := NewBuilder(kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()), true)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.BuildFromAdminKubeconfig(instance, adminKubeconfig())

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newOwnClusterKubeconfig())
	})
}

func TestBuilder_BuildFromAdminKubeconfig_MultipleAdditionalOIDC(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	runtimeResource := &imv1.Runtime{}
	runtimeResource.ObjectMeta.Name = runtimeID
	runtimeResource.ObjectMeta.Namespace = "kcp-system"
	runtimeResource.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = &[]imv1.OIDCConfig{
		{
			OIDCConfig: gardener.OIDCConfig{
				ClientID:  ptr.String(clientID),
				IssuerURL: ptr.String(issuerURL),
			},
		},
		{
			OIDCConfig: gardener.OIDCConfig{
				ClientID:  ptr.String(client2ID),
				IssuerURL: ptr.String(issuer2URL),
			},
		},
	}

	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(runtimeResource).Build()

	t.Run("new kubeconfig was build properly", func(t *testing.T) {
		// given

		builder := NewBuilder(kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()), true)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.BuildFromAdminKubeconfig(instance, adminKubeconfig())

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newOwnClusterKubeconfigWithMultipleContexts())
	})

	t.Run("should return kubeconfig with one context when feature flag is disabled", func(t *testing.T) {
		// given

		builder := NewBuilder(kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()), false)

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.BuildFromAdminKubeconfig(instance, adminKubeconfig())

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newOwnClusterKubeconfig())
	})
}

func skrKubeconfig() *string {
	kc := `
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--ac0d8d9
clusters:
- name: shoot--kyma-dev--ac0d8d9
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--ac0d8d9
  context:
    cluster: shoot--kyma-dev--ac0d8d9
    user: shoot--kyma-dev--ac0d8d9-token
users:
- name: shoot--kyma-dev--ac0d8d9-token
  user:
    token: DKPAe2Lt06a8dlUlE81kaWdSSDVSSf38x5PIj6cwQkqHMrw4UldsUr1guD6Thayw
`
	return &kc
}

func newKubeconfigWithNoUsers() string {
	return `
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--ac0d8d9
clusters:
- name: shoot--kyma-dev--ac0d8d9
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--ac0d8d9
  context:
    cluster: shoot--kyma-dev--ac0d8d9
`
}

func newKubeconfig() string {
	return fmt.Sprintf(`
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--ac0d8d9
clusters:
- name: shoot--kyma-dev--ac0d8d9
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--ac0d8d9
  context:
    cluster: shoot--kyma-dev--ac0d8d9
    user: shoot--kyma-dev--ac0d8d9
users:
- name: shoot--kyma-dev--ac0d8d9
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - "--oidc-issuer-url=%s"
      - "--oidc-client-id=%s"
      - "--oidc-extra-scope=email"
      - "--oidc-extra-scope=openid"
      command: kubectl-oidc_login
      installHint: |
        kubelogin plugin is required to proceed with authentication
        # Homebrew (macOS and Linux)
        brew install int128/kubelogin/kubelogin

        # Krew (macOS, Linux, Windows and ARM)
        kubectl krew install oidc-login

        # Chocolatey (Windows)
        choco install kubelogin
`, issuerURL, clientID,
	)
}

func newKubeconfigWithMultipleContexts() string {
	return fmt.Sprintf(`
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--ac0d8d9
clusters:
- name: shoot--kyma-dev--ac0d8d9
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--ac0d8d9
  context:
    cluster: shoot--kyma-dev--ac0d8d9
    user: shoot--kyma-dev--ac0d8d9
- name: shoot--kyma-dev--ac0d8d9-2
  context:
    cluster: shoot--kyma-dev--ac0d8d9
    user: shoot--kyma-dev--ac0d8d9-2
users:
- name: shoot--kyma-dev--ac0d8d9
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - "--oidc-issuer-url=%s"
      - "--oidc-client-id=%s"
      - "--oidc-extra-scope=email"
      - "--oidc-extra-scope=openid"
      command: kubectl-oidc_login
      installHint: |
        kubelogin plugin is required to proceed with authentication
        # Homebrew (macOS and Linux)
        brew install int128/kubelogin/kubelogin

        # Krew (macOS, Linux, Windows and ARM)
        kubectl krew install oidc-login

        # Chocolatey (Windows)
        choco install kubelogin
- name: shoot--kyma-dev--ac0d8d9-2
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - "--oidc-issuer-url=%s"
      - "--oidc-client-id=%s"
      - "--oidc-extra-scope=email"
      - "--oidc-extra-scope=openid"
      command: kubectl-oidc_login
      installHint: |
        kubelogin plugin is required to proceed with authentication
        # Homebrew (macOS and Linux)
        brew install int128/kubelogin/kubelogin

        # Krew (macOS, Linux, Windows and ARM)
        kubectl krew install oidc-login

        # Chocolatey (Windows)
        choco install kubelogin
`, issuerURL, clientID, issuer2URL, client2ID,
	)
}

func newOwnClusterKubeconfigWithNoUsers() string {
	return `
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--admin
clusters:
- name: shoot--kyma-dev--admin
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--admin
  context:
    cluster: shoot--kyma-dev--admin
`
}

func newOwnClusterKubeconfig() string {
	return fmt.Sprintf(`
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--admin
clusters:
- name: shoot--kyma-dev--admin
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--admin
  context:
    cluster: shoot--kyma-dev--admin
    user: shoot--kyma-dev--admin
users:
- name: shoot--kyma-dev--admin
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - "--oidc-issuer-url=%s"
      - "--oidc-client-id=%s"
      - "--oidc-extra-scope=email"
      - "--oidc-extra-scope=openid"
      command: kubectl-oidc_login
      installHint: |
        kubelogin plugin is required to proceed with authentication
        # Homebrew (macOS and Linux)
        brew install int128/kubelogin/kubelogin

        # Krew (macOS, Linux, Windows and ARM)
        kubectl krew install oidc-login

        # Chocolatey (Windows)
        choco install kubelogin
`, issuerURL, clientID,
	)
}

func newOwnClusterKubeconfigWithMultipleContexts() string {
	return fmt.Sprintf(`
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--admin
clusters:
- name: shoot--kyma-dev--admin
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--admin
  context:
    cluster: shoot--kyma-dev--admin
    user: shoot--kyma-dev--admin
- name: shoot--kyma-dev--admin-2
  context:
    cluster: shoot--kyma-dev--admin
    user: shoot--kyma-dev--admin-2
users:
- name: shoot--kyma-dev--admin
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - "--oidc-issuer-url=%s"
      - "--oidc-client-id=%s"
      - "--oidc-extra-scope=email"
      - "--oidc-extra-scope=openid"
      command: kubectl-oidc_login
      installHint: |
        kubelogin plugin is required to proceed with authentication
        # Homebrew (macOS and Linux)
        brew install int128/kubelogin/kubelogin

        # Krew (macOS, Linux, Windows and ARM)
        kubectl krew install oidc-login

        # Chocolatey (Windows)
        choco install kubelogin
- name: shoot--kyma-dev--admin-2
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - "--oidc-issuer-url=%s"
      - "--oidc-client-id=%s"
      - "--oidc-extra-scope=email"
      - "--oidc-extra-scope=openid"
      command: kubectl-oidc_login
      installHint: |
        kubelogin plugin is required to proceed with authentication
        # Homebrew (macOS and Linux)
        brew install int128/kubelogin/kubelogin

        # Krew (macOS, Linux, Windows and ARM)
        kubectl krew install oidc-login

        # Chocolatey (Windows)
        choco install kubelogin
`, issuerURL, clientID, issuer2URL, client2ID,
	)
}

func adminKubeconfig() string {
	return `
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--admin
clusters:
- name: shoot--kyma-dev--admin
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--admin
  context:
    cluster: shoot--kyma-dev--admin
    user: shoot--kyma-dev--admin-token
users:
- name: shoot--kyma-dev--admin-token
  user:
    token: DKPAe2Lt06a8dlUlE81kaWdSSDVSSf38x5PIj6cwQkqHMrw4UldsUr1guD6Thayw

`
}

func NewFakeKubeconfigProvider(content *string) *fakeKubeconfigProvider {
	return &fakeKubeconfigProvider{
		content: *content,
	}
}

type fakeKubeconfigProvider struct {
	content string
}

func (p *fakeKubeconfigProvider) KubeconfigForRuntimeID(_ string) ([]byte, error) {
	return []byte(p.content), nil
}
