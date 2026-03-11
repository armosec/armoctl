package operator

// CloudFormationTemplate is the embedded CloudFormation template for deploying
// the ARMO ECS Operator. This is a trimmed version containing only the operator
// resources (not the ECS Agent daemon).
const CloudFormationTemplate = `AWSTemplateFormatVersion: "2010-09-09"
Description: >
  ARMO ECS Operator deployment (single instance for cluster visibility).

Parameters:
  Region:
    Type: String
    Description: AWS region for the ECS cluster

  EcsClusterName:
    Type: String
    Description: Name of the existing ECS cluster to deploy into

  CustomerGuid:
    Type: String
    Description: ARMO customer GUID
    NoEcho: true

  AccessKey:
    Type: String
    Description: ARMO API access key
    NoEcho: true

  ApiUrl:
    Type: String
    Description: ARMO backend URL

  EcsOperatorImage:
    Type: String
    Description: Docker image for the ARMO ECS operator

  CloudWatchLogsGroupName:
    Type: String
    Description: >
      CloudWatch Logs Group name for ARMO component logging.
      If the group does not exist it will be created automatically.
      Leave empty to disable logging.
    Default: ""

Conditions:
  LoggingEnabled: !Not [!Equals [!Ref CloudWatchLogsGroupName, ""]]

Resources:
  #############################################################################
  # IAM Roles
  #############################################################################

  EcsTaskExecutionRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: !Sub "armo-ecs-execution-role-${AWS::StackName}"
      AssumeRolePolicyDocument:
        Version: "2012-10-17"
        Statement:
          - Effect: Allow
            Principal:
              Service: ecs-tasks.amazonaws.com
            Action: sts:AssumeRole
      ManagedPolicyArns:
        - arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy
      Policies:
        - PolicyName: CreateLogGroup
          PolicyDocument:
            Version: "2012-10-17"
            Statement:
              - Effect: Allow
                Action:
                  - logs:CreateLogGroup
                Resource: "*"

  EcsOperatorTaskRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: !Sub "armo-ecs-operator-task-role-${AWS::StackName}"
      AssumeRolePolicyDocument:
        Version: "2012-10-17"
        Statement:
          - Effect: Allow
            Principal:
              Service: ecs-tasks.amazonaws.com
            Action: sts:AssumeRole
      Policies:
        - PolicyName: EcsReadOnly
          PolicyDocument:
            Version: "2012-10-17"
            Statement:
              - Effect: Allow
                Action:
                  - ecs:ListClusters
                  - ecs:DescribeClusters
                  - ecs:ListServices
                  - ecs:DescribeServices
                  - ecs:ListTasks
                  - ecs:DescribeTasks
                  - ecs:ListTaskDefinitions
                  - ecs:DescribeTaskDefinition
                  - ecs:ListContainerInstances
                  - ecs:DescribeContainerInstances
                Resource: "*"

  #############################################################################
  # ECS Operator - Task Definition (single instance for cluster visibility)
  #############################################################################

  EcsOperatorTaskDefinition:
    Type: AWS::ECS::TaskDefinition
    Properties:
      Family: armo-ecs-operator
      NetworkMode: host
      Cpu: "512"
      Memory: "1024"
      TaskRoleArn: !GetAtt EcsOperatorTaskRole.Arn
      ExecutionRoleArn: !GetAtt EcsTaskExecutionRole.Arn
      Tags:
        - Key: Purpose
          Value: ARMOEcsOperator
      ContainerDefinitions:
        - Name: ecs-operator
          Image: !Ref EcsOperatorImage
          Essential: true
          Environment:
            - Name: CUSTOMER_GUID
              Value: !Ref CustomerGuid
            - Name: ACCESS_KEY
              Value: !Ref AccessKey
            - Name: BACKEND_URL
              Value: !Ref ApiUrl
          LogConfiguration: !If
            - LoggingEnabled
            - LogDriver: awslogs
              Options:
                awslogs-group: !Ref CloudWatchLogsGroupName
                awslogs-create-group: "true"
                awslogs-region: !Ref Region
                awslogs-stream-prefix: ecs-operator
            - !Ref "AWS::NoValue"

  #############################################################################
  # ECS Operator - Service (single replica)
  #############################################################################

  EcsOperatorService:
    Type: AWS::ECS::Service
    Properties:
      ServiceName: armo-ecs-operator
      Cluster: !Ref EcsClusterName
      TaskDefinition: !Ref EcsOperatorTaskDefinition
      DesiredCount: 1
      SchedulingStrategy: REPLICA
      DeploymentConfiguration:
        DeploymentCircuitBreaker:
          Enable: true
          Rollback: true
        MaximumPercent: 200
        MinimumHealthyPercent: 100

Outputs:
  EcsOperatorServiceArn:
    Description: ARN of the ECS Operator service
    Value: !Ref EcsOperatorService

  EcsOperatorTaskDefinitionArn:
    Description: ARN of the ECS Operator task definition
    Value: !Ref EcsOperatorTaskDefinition
`
