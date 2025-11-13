import { Construct } from 'constructs';
import { LogGroup } from 'aws-cdk-lib/aws-logs'
import { DockerImageAsset } from 'aws-cdk-lib/aws-ecr-assets';
import { ContainerImage, FargatePlatformVersion } from 'aws-cdk-lib/aws-ecs';

import * as cdk from 'aws-cdk-lib/core';
import * as path from 'path';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as ecs from 'aws-cdk-lib/aws-ecs';
import * as iam from 'aws-cdk-lib/aws-iam'
import * as ssm from 'aws-cdk-lib/aws-ssm'
import * as ecs_patterns from "aws-cdk-lib/aws-ecs-patterns";

export class FargateStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const asset = new DockerImageAsset(this, 'ddb-local-image', {
      directory: path.join(__dirname, "..", ".."),
    });

    const vpc = new ec2.Vpc(this, "EcsVpc", {
      maxAzs: 3 // Default is all AZs in region
    });

    const cluster = new ecs.Cluster(this, "TestCluster", {
      vpc: vpc,
      clusterName: "ddb-local-cluster",
      containerInsights: false
    });

    const group = new LogGroup(this, "FargateLogGroup", {
      logGroupName: "/ecs/ddb-local-service"
    })

    const task = new ecs.FargateTaskDefinition(this, "MyTask", {
      cpu: 512,
      memoryLimitMiB: 1024,
    })

    const container = new ecs.ContainerDefinition(this, "MyContainer", {
      image: ContainerImage.fromDockerImageAsset(asset),
      taskDefinition: task,
      environment: {
        PARAM1: "test1"
      },
      logging: new ecs.AwsLogDriver({
        logGroup: group,
        streamPrefix: `ddb-local-service`,
      })
    }
    )

    const service = new ecs_patterns.ApplicationLoadBalancedFargateService(this, "MyFargateService", {
      cluster: cluster, // Required
      cpu: 512, // Default is 256
      desiredCount: 2, // Default is 1
      taskImageOptions: { image: ecs.RepositoryImage.fromDockerImageAsset(asset) },
      memoryLimitMiB: 2048, // Default is 512
      publicLoadBalancer: true // Default is false
    });

    const arn = ssm.StringParameter.valueForStringParameter(this, "/ddblocal/tableArn")

    service.taskDefinition.addToTaskRolePolicy(new iam.PolicyStatement({
      actions:["dynamodb:PutItem","dynamodb:GetItem"],
      resources:[arn],
      effect: iam.Effect.ALLOW
    }))

    const dns = new ssm.StringParameter( this,"serviceDNS",{
      parameterName: "local-ddb-service-dns",
      stringValue: service.loadBalancer.loadBalancerDnsName
    })
  }
}
