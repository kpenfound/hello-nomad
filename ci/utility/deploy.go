package utility

import (
	"context"
	"os"
	"time"

	"dagger.io/dagger"
	"github.com/hashicorp/nomad/api"
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

func DeployNomadJob(ctx context.Context, imageref string) error {
	cli, err := api.NewClient(&api.Config{Address: os.Getenv("NOMAD_ADDR")})
	if err != nil {
		return err
	}

	job := getJob(imageref)
	_, _, err = cli.Jobs().Register(job, &api.WriteOptions{})
	return err
}

func getJob(imageref string) *api.Job {
	name := "hello"
	count := 5
	portLabel := "http"
	restartAttempts := 2
	restartInterval := time.Minute * 30
	restartDelay := time.Second * 15
	restartMode := "fail"
	return &api.Job{
		ID:          &name,
		Name:        &name,
		Datacenters: []string{"dc1"},
		TaskGroups: []*api.TaskGroup{
			{
				Name:  &name,
				Count: &count,
				Tasks: []*api.Task{
					{
						Name:   "hello-server",
						Driver: "docker",
						Config: map[string]interface{}{
							"image": imageref,
							"ports": []string{"http"},
						},
					},
				},
				Networks: []*api.NetworkResource{
					{
						DynamicPorts: []api.Port{
							{
								Label: portLabel,
								To:    8080,
							},
						},
					},
				},
				Services: []*api.Service{
					{
						Name:      "hello-server",
						Tags:      []string{"urlprefix-/"},
						PortLabel: portLabel,
						Checks: []api.ServiceCheck{
							{
								Name:     "alive",
								Type:     "http",
								Path:     "/",
								Interval: time.Second * 10,
								Timeout:  time.Second * 2,
							},
						},
					},
				},
				RestartPolicy: &api.RestartPolicy{
					Interval: &restartInterval,
					Delay:    &restartDelay,
					Mode:     &restartMode,
					Attempts: &restartAttempts,
				},
			},
		},
	}
}

func DeployNetlify(client *dagger.Client, ctx context.Context, build *dagger.Directory) error {
	site := GetVaultSecret(client, "netlify_site")
	token := GetVaultSecret(client, "netlify_token")

	netlify := client.Container().
		From("node:16").
		Exec(dagger.ContainerExecOpts{
			Args: []string{"npm", "install", "netlify-cli", "-g"},
		}).
		WithMountedDirectory("/build", build).
		WithSecretVariable("NETLIFY_SITE_ID", site).
		WithSecretVariable("NETLIFY_AUTH_TOKEN", token).
		Exec(dagger.ContainerExecOpts{
			Args: []string{"netlify", "deploy", "--dir", "/build"},
		})

	_, err := netlify.ExitCode(ctx)
	return err
}
