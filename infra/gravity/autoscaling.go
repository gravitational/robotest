package gravity

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gravitational/robotest/infra/ops"
	"github.com/gravitational/trace"
)

func (c *TestContext) AutoScale(target int) ([]Gravity, error) {
	c.Logger().Debug("attempting to connect to AWS api")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.provisionerCfg.Ops.Region),
		Credentials: credentials.NewStaticCredentials(c.provisionerCfg.Ops.AccessKey, c.provisionerCfg.Ops.SecretKey, ""),
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	svc := autoscaling.New(sess)

	// first, let's set the desired capacity
	setCapacity := &autoscaling.SetDesiredCapacityInput{
		AutoScalingGroupName: aws.String(c.provisionerCfg.clusterName),
		DesiredCapacity:      aws.Int64(int64(target)),
		HonorCooldown:        aws.Bool(false),
	}
	_, err = svc.SetDesiredCapacity(setCapacity)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	nodes := []Gravity{}

	// next we need to find all the instances that were just created, and build objects and ssh connections to them
	describeASG := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(c.provisionerCfg.clusterName),
		},
	}
	result, err := svc.DescribeAutoScalingGroups(describeASG)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	if len(result.AutoScalingGroups) != 1 {
		return nil, trace.Errorf("unexpected number of autoscaling groups found: 1 != %v", len(result.AutoScalingGroups))
	}
	if len(result.AutoScalingGroups[0].Instances) != target {
		return nil, trace.Errorf("unexpected number of instances in autoscaling group %v != %v", target, len(result.AutoScalingGroups[0].Instances))
	}

	for _, instance := range result.AutoScalingGroups[0].Instances {
		// attempt to get the actual instance
		node, err := c.getAWSNodes(sess, "instance-id", *instance.InstanceId)
		if err != nil {
			return nil, trace.Wrap(err)
		}

		if len(node) != 1 {
			return nil, trace.Errorf("unexpected number of aws nodes found. 1 != %v", len(nodes))
		}
		nodes = append(nodes, node[0])

	}

	return nodes, nil
}

func (c *TestContext) getAWSNodes(sess *session.Session, filterName string, filterValue string) ([]Gravity, error) {
	cloudParams, err := makeDynamicParams(c.provisionerCfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	ec2svc := ec2.New(sess)
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(filterName),
				Values: []*string{aws.String(filterValue)},
			},
		},
	}

	resp, err := ec2svc.DescribeInstances(params)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var nodes []Gravity
	for _, reservation := range resp.Reservations {
		for _, inst := range reservation.Instances {
			node := ops.New(*inst.PublicIpAddress, *inst.PrivateIpAddress, c.provisionerCfg.Ops.SSHUser, c.provisionerCfg.Ops.SSHKeyPath)

			gravityNode, err := configureVM(c.Context(), c.Logger(), node, *cloudParams)
			if err != nil {
				return nil, trace.Wrap(err)
			}

			nodes = append(nodes, gravityNode)
		}
	}
	return nodes, nil
}
