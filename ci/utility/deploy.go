package utility

import (
	"context"

	"dagger.io/dagger"
)

const (
	publishAddr = "kylepenfound/hello-nomad:latest"
)

func PublishImage(client *dagger.Client, ctx context.Context, variants []*dagger.Container) (string, error) {
	return client.Container().Publish(
		ctx,
		publishAddr,
		dagger.ContainerPublishOpts{
			PlatformVariants: variants,
		})
}

func Deploy(ctx context.Context, imageref string) error {
	return nil
}
