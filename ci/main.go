package main

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
	"github.com/kpenfound/hello-nomad/ci/utility"
)

func main() {
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Backend CICD
	err = backendBuildAndDeploy(ctx, client)
	if err != nil {
		panic(err)
	}
	fmt.Println("Updated hello-nomad job")

	// Frontend CICD
	err = frontendBuildAndDeploy(ctx, client)
	if err != nil {
		panic(err)
	}
	fmt.Println("Netlify site deployed")
}

func backendBuildAndDeploy(ctx context.Context, client *dagger.Client) error {
	// get backend project to build
	project := utility.GetBackend(client)

	// Multiplatform image for amd64+arm64
	platformToArch := map[dagger.Platform]string{
		"linux/amd64": "amd64",
		"linux/arm64": "arm64",
	}
	variants := make([]*dagger.Container, 0, len(platformToArch))

	// Build image for each platform
	for platform, arch := range platformToArch {
		build := utility.AppBuild(client, project, platform, arch)
		variants = append(variants, build)
	}

	// publish image
	addr, err := utility.PublishImage(client, ctx, variants)
	if err != nil {
		return err
	}

	// deploy job
	return utility.DeployNomadJob(ctx, addr)
}

func frontendBuildAndDeploy(ctx context.Context, client *dagger.Client) error {
	project := utility.GetFrontend(client)
	return utility.DeployNetlify(client, ctx, project)
}
