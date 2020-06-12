/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gravity

import (
	"github.com/gravitational/robotest/infra/providers/ops"
	"github.com/gravitational/robotest/lib/wait"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gravitational/trace"
)

// AutoScale will update the autoscaling group to the target number of nodes,
// and return a new list of nodes to be used for testing
func (c *TestContext) AutoScale(target int) ([]Gravity, error) {
	c.Logger().Debug("attempting to connect to AWS api")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.provisionerCfg.Ops.EC2Region),
		Credentials: credentials.NewStaticCredentials(c.provisionerCfg.Ops.EC2AccessKey, c.provisionerCfg.Ops.EC2SecretKey, ""),
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
	c.Logger().WithField("target_count", setCapacity).Debug("setting scaling group desired capacity")
	_, err = svc.SetDesiredCapacity(setCapacity)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// next we need to find all the instances that were just created, and build objects and ssh connections to them
	describeASG := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(c.provisionerCfg.clusterName),
		},
	}

	// we may need to wait for the nodes to get assigned by the autoscaling group
	// so we need to repeat our API requests until we get the expected nodes
	nodes := []Gravity{}
	retryer := wait.Retryer{
		Delay:    autoscaleWait,
		Attempts: autoscaleRetries,
	}

	var result *autoscaling.DescribeAutoScalingGroupsOutput
	err = retryer.Do(c.Context(), func() (err error) {
		result, err = checkForNodeAssignment(svc, describeASG, target)
		return trace.Wrap(err)
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	ec2svc := ec2.New(sess)
	for _, instance := range result.AutoScalingGroups[0].Instances {
		// attempt to get the actual instance for each instance-id in the cluster
		node, err := c.getAWSNodes(ec2svc, "instance-id", *instance.InstanceId)
		if err != nil {
			return nil, trace.Wrap(err)
		}

		if len(node) != 1 {
			return nil, trace.BadParameter("unexpected number of AWS nodes found. 1 != %v", len(nodes))
		}
		nodes = append(nodes, node[0])

	}

	return nodes, nil
}

// checkForNodeAssignment will check to see if our auto-scaling group has the requested number of nodes assigned
func checkForNodeAssignment(svc *autoscaling.AutoScaling, describeASG *autoscaling.DescribeAutoScalingGroupsInput, target int) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	result, err := svc.DescribeAutoScalingGroups(describeASG)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	if len(result.AutoScalingGroups) != 1 {
		return nil, trace.BadParameter("unexpected number of autoscaling groups found: 1 != %v", len(result.AutoScalingGroups))
	}
	if len(result.AutoScalingGroups[0].Instances) != target {
		return nil, trace.BadParameter("unexpected autoscaling count of instances. expected: %v got: %v", target, len(result.AutoScalingGroups[0].Instances))
	}
	return result, nil
}

// getAWSNodes will connect to the AWS API, and get a listing of nodes matching the specified filter.
func (c *TestContext) getAWSNodes(ec2svc *ec2.EC2, filterName string, filterValue string) (nodes []*gravity, err error) {
	cloudParams, err := makeDynamicParams(c.provisionerCfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

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

	for _, reservation := range resp.Reservations {
		for _, inst := range reservation.Instances {
			node := ops.New(*inst.PublicIpAddress, *inst.PrivateIpAddress,
				c.provisionerCfg.Ops.SSHUser, c.provisionerCfg.Ops.SSHKeyPath)

			gravityNode, err := connectVM(c.Context(), c.Logger(), node, *cloudParams)
			if err != nil {
				return nil, trace.Wrap(err)
			}
			gravityNode.installDir = "/bin"

			err = configureVM(c.Context(), c.Logger(), gravityNode, *cloudParams)
			if err != nil {
				return nil, trace.Wrap(err)
			}

			nodes = append(nodes, gravityNode)
		}
	}
	return nodes, nil
}
