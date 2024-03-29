package utility

import (
	"fmt"
	"os"
	"time"

	"dagger.io/dagger"
)

const (
	appSecrets   = "kv/hello-nomad" // TODO : different path for each env/role
	vaultTlsName = "tycho.belt"
)

func AddAppSecrets(client *dagger.Client, container *dagger.Container) *dagger.Container {
	return container
}

// getEnvSecret
// gets a value from the environment and returns it as a dagger.Secret
func GetEnvSecret(client *dagger.Client, name string) *dagger.Secret {
	return client.Host().EnvVariable(name).Secret()
}

// getVaultSecret
// gets a value from Vault and returns it as a dagger.Secret
func GetVaultSecret(client *dagger.Client, name string) *dagger.Secret {
	// Get Vault TLS certs
	vaultCaCert := client.Host().Directory("ci", dagger.HostDirectoryOpts{Include: []string{"cacert.pem"}})

	// Get secret from Vault
	vault := client.Container().
		From("hashicorp/vault:latest").
		WithEnvVariable("NOCACHE", time.Now().String()). // never reuse cache
		WithEnvVariable("SKIP_SETCAP", "true").
		WithEnvVariable("VAULT_TOKEN", os.Getenv("VAULT_TOKEN")). // Vault authentication
		WithEnvVariable("VAULT_ADDR", os.Getenv("VAULT_ADDR")).
		WithEnvVariable("VAULT_TLS_SERVER_NAME", vaultTlsName). // Vault TLS
		WithMountedDirectory("/tls", vaultCaCert).
		WithEnvVariable("VAULT_CACERT", "/tls/cacert.pem").
		WithExec([]string{"vault", "kv", "get", fmt.Sprintf("-field=%s", name), appSecrets},
			dagger.ContainerWithExecOpts{RedirectStdout: "/mysecret"})

	fmt.Printf("Got %s from vault!\n", name)
	// return Vault result as Secret
	return vault.File("/mysecret").Secret()
}
