import { Construct } from 'constructs';

import * as cdk from 'aws-cdk-lib/core';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb'
import * as ssm from 'aws-cdk-lib/aws-ssm'

export class DynamoStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

   const table = new dynamodb.Table(this, 'PostTable', {
      tableName: "blog-post-table",
      partitionKey: { name: 'id', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      //removalPolicy: cdk.RemovalPolicy.RETAIN,
      //pointInTimeRecovery: true
    });

    const name = new ssm.StringParameter(this, 'DDBNameParam', {
      description: 'DDB Table Name',
      parameterName: "/ddblocal/tableName",
      stringValue: table.tableName,
      tier: ssm.ParameterTier.STANDARD,
    });

    const arn = new ssm.StringParameter(this, 'DDBArnParam', {
      description: 'DDB Table Arn',
      parameterName: "/ddblocal/tableArn",
      stringValue: table.tableArn,
      tier: ssm.ParameterTier.STANDARD,
    });
  }
}
