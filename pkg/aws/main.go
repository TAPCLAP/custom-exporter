package aws

import (
	"fmt"
	"net"
	"os"
	"sync"
    "strings"
    // "encoding/json"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

const (
	audience = "https://iam.api.cloud.yandex.net/iam/v1/tokens"
)

var (
	updateServerList sync.Mutex
)

type Server struct {
	ID             string	`json:"id"`
	Name           string	`json:"name"`
    PrivateDnsName string   `json:"private_dns_name"`
	Type           string	`json:"type"`
	Zone           string   `json:"zone"`
	Region         string	`json:"region"`
	PublicIP       net.IP   `json:"public_ip"`
	PrivateIP      net.IP   `json:"private_ip"`

}

type ClientConfig struct {
	Region string
	AccessKeyID string
	SecretAccessKey string
}

type Client struct {
	region string
	client *session.Session
}

type AWSClouds struct {
	servers []Server
	clients []Client
}


func NewAwsClouds(clientClouds []ClientConfig) (*AWSClouds) {
	a := AWSClouds{}
	for _, c := range clientClouds {
        sess, err := session.NewSession(&aws.Config{
            Region: aws.String(c.Region),
            Credentials: credentials.NewStaticCredentials(c.AccessKeyID, c.SecretAccessKey, ""),
        })
        if err != nil {
            fmt.Fprintf(os.Stderr, "aws.NewAwsClouds: error creating session: %v\n", err)
            continue
        }

		a.clients = append(a.clients, Client{
			region: c.Region,
			client: sess,
		})

	}
	return &a
}


func (a *AWSClouds) getServers() {
    for _, c := range a.clients {
        svc := ec2.New(c.client)
        input := &ec2.DescribeInstancesInput{}
        result, err := svc.DescribeInstances(input)
        if err != nil {
            fmt.Fprintf(os.Stderr, "aws.getServers: error describing instances: %v\n", err)
            continue
        }

        for _, reservation := range result.Reservations {
            for _, instance := range reservation.Instances {
                // data, _ := json.MarshalIndent(instance, "", "    ")
                // fmt.Println(string(data))

                publicIP := ""
                privateIP := ""
                if instance.PublicIpAddress != nil {
                    publicIP = *instance.PublicIpAddress
                }
                if instance.PrivateIpAddress != nil {
                    privateIP = *instance.PrivateIpAddress
                }
                privateDnsName := *instance.PrivateDnsName
                name := *instance.PrivateDnsName
                for _, t := range instance.Tags {
                    if strings.Contains(*t.Key, "aws:eks") {
                        break
                    }
                    if *t.Key == "Name" {
                        name = *t.Value
                        break
                    }

                }
				updateServerList.Lock()
                a.servers = append(a.servers, Server{
                    ID:             *instance.InstanceId,
                    Name:           name,
                    PrivateDnsName: privateDnsName,
                    Type:           *instance.InstanceType,
                    Zone:           *instance.Placement.AvailabilityZone,
                    Region:         c.region,
                    PublicIP:       net.ParseIP(publicIP),
                    PrivateIP:      net.ParseIP(privateIP),
                })
				updateServerList.Unlock()
            }
        }
    }
}

func (a *AWSClouds) GetServers() []Server {
	a.getServers()
	return a.servers
}
