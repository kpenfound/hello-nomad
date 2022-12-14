package utility

import "dagger.io/dagger"

func GetBackend(client *dagger.Client) *dagger.Directory {
	return client.Host().Directory(".", dagger.HostDirectoryOpts{
		Exclude: []string{
			"ci/",
			".vscode",
			".git",
			".gitignore",
			"README",
			"website",
		},
	})
}

func GetFrontend(client *dagger.Client) *dagger.Directory {
	return client.Host().Directory("website")
}

func AppBuild(client *dagger.Client, project *dagger.Directory, platform dagger.Platform, arch string) *dagger.Container {
	greeting := GetVaultSecret(client, "greeting")

	builder := client.Container().
		From("golang:1.19").
		WithMountedDirectory("/src", project).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", arch).
		WithSecretVariable("SECRET_GREETING", greeting).
		WithExec([]string{"go", "test"}).
		WithExec([]string{"sh", "-c", "go build -ldflags \"-X main.SecretGreeting=$SECRET_GREETING\" -o hello"})

	// Build container on production base with build artifact
	base := client.Container(dagger.ContainerOpts{Platform: platform}).
		From("alpine").
		WithFile("/bin/hello", builder.File("/src/hello")).
		WithEntrypoint([]string{"/bin/hello"})
	return base
}
