package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"dagger.io/dagger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

var platformToArch = map[dagger.Platform]string{
	"linux/amd64": "amd64",
	"linux/arm64": "arm64",
}

func main() {
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// get project dir
	project := client.Host().Workdir()

	variants := make([]*dagger.Container, 0, len(platformToArch))
	for platform, arch := range platformToArch {
		// assemble golang build
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
		// add built container to container variants
		variants = append(variants, base)
	}
	addr, err := client.Container().Publish(
		ctx,
		"public.ecr.aws/t5t3s6c1/hello:dev",
		dagger.ContainerPublishOpts{
			PlatformVariants: variants,
		})
	if err != nil {
		panic(err)
	}

	fmt.Println(addr)
	err = deploy(ctx, addr)
	if err != nil {
		fmt.Printf("Error deploying hello-eks: %v", err)
	}
	fmt.Println("Updated hello-eks deployment")
}

func deploy(ctx context.Context, imageref string) error {
	// get kube client
	clientset, err := getKubeClient(ctx)
	if err != nil {
		return err
	}
	// get pod or service?
	return rollingDeployment(ctx, clientset, imageref)
}

func rollingDeployment(ctx context.Context, clientset *kubernetes.Clientset, imageref string) error {
	deployments := clientset.AppsV1().Deployments("default")

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := deployments.Get(ctx, "hello-eks", metav1.GetOptions{})
		if err != nil {
			return err
		}

		result.Spec.Template.Spec.Containers[0].Image = imageref
		_, err = deployments.Update(ctx, result, metav1.UpdateOptions{})
		return err
	})
}

// With help from https://stackoverflow.com/questions/60547409/unable-to-obtain-kubeconfig-of-an-aws-eks-cluster-in-go-code/60573982#60573982
func getKubeClient(ctx context.Context) (*kubernetes.Clientset, error) {
	// Get EKS service
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	eksSvc := eks.New(sess)

	// Get cluster
	input := &eks.DescribeClusterInput{
		Name: aws.String("hello-eks"),
	}
	cluster, err := eksSvc.DescribeCluster(input)
	if err != nil {
		return nil, fmt.Errorf("Error calling DescribeCluster: %v", err)
	}
	// Get token
	gen, err := token.NewGenerator(true, false)
	if err != nil {
		return nil, err
	}
	opts := &token.GetTokenOptions{
		ClusterID: aws.StringValue(cluster.Cluster.Name),
	}
	tok, err := gen.GetWithOptions(opts)
	if err != nil {
		return nil, err
	}
	// b64 decode CA
	ca, err := base64.StdEncoding.DecodeString(aws.StringValue(cluster.Cluster.CertificateAuthority.Data))
	if err != nil {
		return nil, err
	}
	// create k8s clientset
	return kubernetes.NewForConfig(
		&rest.Config{
			Host:        aws.StringValue(cluster.Cluster.Endpoint),
			BearerToken: tok.Token,
			TLSClientConfig: rest.TLSClientConfig{
				CAData: ca,
			},
		},
	)
}
