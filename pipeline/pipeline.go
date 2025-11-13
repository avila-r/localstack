package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awskinesis"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/constructs-go/constructs/v10"
)

type PipelineStackProps struct {
	awscdk.StackProps
	Source string
	Dist   string
}

func String(s string) *string {
	return &s
}

func CreatePipelineStack(scope constructs.Construct, id string, props PipelineStackProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, &id, &props.StackProps)

	vpc := awsec2.NewVpc(stack, String("VPC"), &awsec2.VpcProps{})

	awsec2.NewGatewayVpcEndpoint(stack, String("DynamoEndpoint"), &awsec2.GatewayVpcEndpointProps{
		Vpc:     vpc,
		Service: awsec2.InterfaceVpcEndpointAwsService_KINESIS_STREAMS(),
	})

	stream := awskinesis.NewStream(stack, String("Kinesis"), &awskinesis.StreamProps{
		StreamName: String("KinesisStream"),
	})

	table := awsdynamodb.NewTable(stack, String("DynamoDBTable"), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: String("id"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		Stream: awsdynamodb.StreamViewType_NEW_AND_OLD_IMAGES,
	})

	type (
		LambdaIdentifier    string
		LambdaConfiguration map[string]*string
	)

	url := String("http://localhost.localstack.cloud:4566")

	configuration := map[LambdaIdentifier]LambdaConfiguration{
		"upstream": {
			"LAMBDA_STAGE":     String("upstream"),
			"STREAM_NAME":      stream.StreamName(),
			"AWS_ENDPOINT_URL": url,
		},
		"midstream": {
			"LAMBDA_STAGE":     String("midstream"),
			"TABLE_NAME":       table.TableName(),
			"AWS_ENDPOINT_URL": url,
		},
		"downstream": {
			"LAMBDA_STAGE":     String("downstream"),
			"AWS_ENDPOINT_URL": url,
		},
	}

	lambdas := make(map[LambdaIdentifier]awslambda.IFunction)
	for identifier, properties := range configuration {
		environment := map[string]*string{
			"GOCACHE": String("/tmp/go-cache"),
			"GOOS":    String("linux"),
			"GOARCH":  String("amd64"),
		}

		code := awslambda.Code_FromAsset(String(filepath.Join(props.Source, string(identifier))), &awss3assets.AssetOptions{
			Bundling: &awscdk.BundlingOptions{
				Image:       awscdk.DockerImage_FromRegistry(String("golang:1.21")),
				Command:     &[]*string{String("bash"), String("-c"), String("go build -o /asset-output/bootstrap .")},
				Environment: &environment,
			},
		})

		env := (map[string]*string)(properties)

		lambdas[identifier] = awslambda.NewFunction(stack, String("Lambda"+strings.ToTitle(string(identifier))), &awslambda.FunctionProps{
			Vpc:          vpc,
			Runtime:      awslambda.Runtime_PROVIDED_AL2(),
			Code:         code,
			Handler:      String("bootstrap"),
			Environment:  &env,
			Architecture: awslambda.Architecture_X86_64(),
		})
	}

	api := awsapigateway.NewRestApi(stack, String("Api"), &awsapigateway.RestApiProps{
		DefaultIntegration: awsapigateway.NewLambdaIntegration(lambdas["upstream"], nil),
		Description:        String("API Gateway for Serverless Data Processing Pipeline"),
	})

	method := api.Root().AddMethod(String("POST"), awsapigateway.NewLambdaIntegration(lambdas["upstream"], nil), nil)

	stream.GrantWrite(lambdas["upstream"].Role())
	stream.GrantRead(lambdas["midstream"].Role())

	table.GrantWriteData(lambdas["midstream"].Role())
	table.GrantStreamRead(lambdas["downstream"].Role())

	lambdas["downstream"].Role().AddToPrincipalPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			String("cloudwatch:PutMetricData"),
		},
		Resources: &[]*string{
			String("*"),
		},
	}))

	lambdas["midstream"].AddEventSource(awslambdaeventsources.NewKinesisEventSource(stream, &awslambdaeventsources.KinesisEventSourceProps{
		StartingPosition: awslambda.StartingPosition_LATEST,
	}))

	lambdas["downstream"].AddEventSource(awslambdaeventsources.NewDynamoEventSource(table, &awslambdaeventsources.DynamoEventSourceProps{
		StartingPosition: awslambda.StartingPosition_LATEST,
	}))

	endpoint := fmt.Sprintf("%s%s", *api.Url(), *method.PhysicalName())

	awscdk.NewCfnOutput(stack, String("ApiGatewayMethodEndpoint"), &awscdk.CfnOutputProps{
		Value: &endpoint,
	})

	awscdk.NewCfnOutput(stack, String("KinesisStreamName"), &awscdk.CfnOutputProps{
		Value: stream.StreamName(),
	})

	awscdk.NewCfnOutput(stack, String("DynamoDBTableName"), &awscdk.CfnOutputProps{
		Value: table.TableName(),
	})

	awscdk.NewCfnOutput(stack, String("Environment"), &awscdk.CfnOutputProps{
		Value: String("LocalStack"),
	})

	return stack
}

func main() {
	app := awscdk.NewApp(nil)

	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	env := func(path, fallback string) string {
		value := os.Getenv(path)
		if value == "" {
			return fallback
		}
		return value
	}

	CreatePipelineStack(app, "ServerlessDataProcessingPipelineStack", PipelineStackProps{
		StackProps: awscdk.StackProps{
			Env: nil,
		},
		Dist: func() string {
			return filepath.Join(root, env("LAMBDA_DIST_PATH", "lambda/dist"))
		}(),
		Source: func() string {
			return filepath.Join(root, env("LAMBDA_SRC_PATH", "lambda/src"))
		}(),
	})

	app.Synth(nil)
}
