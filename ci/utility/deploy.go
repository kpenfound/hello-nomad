package utility

import (
	"context"
	"time"

	"dagger.io/dagger"
	"github.com/hashicorp/nomad/api"
)

const (
	publishAddr = "kylepenfound/hello-nomad:latest"
	nomadAddr   = "http://tycho.belt:4646"
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
	cli, err := api.NewClient(&api.Config{Address: nomadAddr})
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
