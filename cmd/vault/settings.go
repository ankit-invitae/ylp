package vault

type VaultConfig struct {
	EnvName   string
	Addr      string
	VaultUrls []string
}

func NewVaultConfig(envName string) VaultConfig {
	if envName == "prod" {
		return getProdSetting()
	}
	return getDevSetting()
}

func getDevSetting() VaultConfig {
	return VaultConfig{
		EnvName: "dev",
		Addr:    "https://vault.dev.locusdev.net/",
		VaultUrls: []string{
			"secret/lims/berossus",
			"secret/lims/berossusTest",
		},
	}
}

func getProdSetting() VaultConfig {
	return VaultConfig{
		EnvName: "prod",
		Addr:    "https://vault.prod.locusdev.net/",
		VaultUrls: []string{
			"secret/lims/berossus",
			"secret/lims/berossus2",
			"secret/lims/berossus-dlo",
			"secret/lims/berossus-pipe",
		},
	}
}
