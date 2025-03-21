package provider

import (
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/stretchr/testify/assert"
)

func TestGcpTrialInput_ApplyParametersWithRegion(t *testing.T) {
	// given
	svc := GcpTrialInput{
		PlatformRegionMapping: map[string]string{
			"cf-eu": "europe",
		},
	}

	// when
	t.Run("use platform region mapping", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-eu",
		})

		//then
		assert.Equal(t, "europe-west3", input.GardenerConfig.Region)
	})

	// when
	t.Run("use customer mapping", func(t *testing.T) {
		// given
		input := svc.Defaults()
		us := "us"

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-eu",
			Parameters: pkg.ProvisioningParametersDTO{
				Region: &us,
			},
		})

		//then
		assert.Equal(t, "us-central1", input.GardenerConfig.Region)
	})

	// when
	t.Run("use default region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{})

		//then
		assert.Equal(t, "europe-west3", input.GardenerConfig.Region)
	})

	// when
	t.Run("use default region for not defined mapping", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-southamerica",
		})

		//then
		assert.Equal(t, "europe-west3", input.GardenerConfig.Region)
	})

	// when
	t.Run("use default region for Assured Workloads", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-sa30",
		})

		//then
		assert.Equal(t, DefaultGCPAssuredWorkloadsRegion, input.GardenerConfig.Region)
	})
}

func TestGcpInput_SingleZone_ApplyParameters(t *testing.T) {
	// given
	svc := GcpInput{}

	// when
	t.Run("zones with default region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{},
		})

		// then
		assert.Equal(t, "europe-west3", input.GardenerConfig.Region)
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, 1)
		assert.Subset(t, []string{"europe-west3-a", "europe-west3-b", "europe-west3-c"}, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones)
	})

	// when
	t.Run("zones with specified region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{
				Region: ptr.String("us-central1"),
			},
		})

		// then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, 1)
		assert.Subset(t, []string{"us-central1-a", "us-central1-b", "us-central1-c"}, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones)
	})

	// when
	t.Run("use default region and default zones count for Assured Workloads", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-sa30",
		})

		// then
		assert.Equal(t, DefaultGCPAssuredWorkloadsRegion, input.GardenerConfig.Region)
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, 1)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones {
			regionFromZone := zone[:len(zone)-2]
			assert.Equal(t, DefaultGCPAssuredWorkloadsRegion, regionFromZone)
		}
	})
}

func TestGcpInput_MultiZone_ApplyParameters(t *testing.T) {
	// given
	svc := GcpInput{
		MultiZone:                    true,
		ControlPlaneFailureTolerance: "zone",
	}

	// when
	t.Run("zones with default region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{},
		})

		// then
		assert.Equal(t, "europe-west3", input.GardenerConfig.Region)
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, 3)
		assert.Subset(t, []string{"europe-west3-a", "europe-west3-b", "europe-west3-c"}, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones)
		assert.Equal(t, "zone", *input.GardenerConfig.ControlPlaneFailureTolerance)
	})

	// when
	t.Run("zones with specified region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{
				Region: ptr.String("us-central1"),
			},
		})

		// then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, 3)
		assert.Subset(t, []string{"us-central1-a", "us-central1-b", "us-central1-c"}, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones)
		assert.Equal(t, "zone", *input.GardenerConfig.ControlPlaneFailureTolerance)
	})

	// when
	t.Run("use default region and default zones count for Assured Workloads", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-sa30",
		})

		// then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, 3)
		assert.Subset(t, []string{"me-central2-a", "me-central2-b", "me-central2-c"}, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones)
		assert.Equal(t, "zone", *input.GardenerConfig.ControlPlaneFailureTolerance)
	})
}
