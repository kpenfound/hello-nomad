package utility

import "dagger.io/dagger"

func GetProject(client *dagger.Client) *dagger.Directory {
	return client.Host().Workdir(dagger.HostWorkdirOpts{
		Exclude: []string{
			"ci/",
			".vscode",
			".git",
			".gitignore",
			"README",
		},
	})
}

func AppBuild(client *dagger.Client, project *dagger.Directory, platform dagger.Platform, arch string) *dagger.Container {
	builder := client.Container().
		From("golang:latest").
		WithMountedDirectory("/src", project).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", arch).
		Exec(dagger.ContainerExecOpts{
			Args: []string{"go", "build", "-o", "hello"},
		})

	// Build container on production base with build artifact
	base := client.Container(dagger.ContainerOpts{Platform: platform}).
		From("alpine")
	// copy build artifact from builder image
	base = base.WithFS(
		base.FS().WithFile("/bin/hello",
			builder.File("/src/hello"),
		)).
		WithEntrypoint([]string{"/bin/hello"})
	return base
}
