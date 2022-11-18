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

	secret := utility.GetVaultSecret(client, "target")
	txt, err := secret.Plaintext(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Secret is: %s\n", txt)

	// get project to build
	project := utility.GetProject(client)

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
		panic(err)
	}
	fmt.Println(addr)

	// // deploy job
	// err = deploy(ctx, addr)
	// if err != nil {
	// 	panic(err)
	// }

	fmt.Println("Updated hello-nomad job")
}
