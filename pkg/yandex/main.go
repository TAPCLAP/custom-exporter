package yandex

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"sync"

	jwt "github.com/golang-jwt/jwt/v4"
	yandex "github.com/yandex-cloud/go-sdk"
	compute "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
)

const (
	audience = "https://iam.api.cloud.yandex.net/iam/v1/tokens"
)

var (
	updateServerList sync.Mutex
)

type Server struct {
	ID         string	`json:"id"`
	Name       string	`json:"name"`
	Type       string	`json:"type"`
	Zone       string   `json:"zone"`
	Region     string	`json:"region"`
	PublicIP   net.IP   `json:"public_ip"`
	PrivateIP  net.IP   `json:"private_ip"`
	CpuCount   uint8    `json:"cpu_count"`
	Memory     uint64   `json:"memory"`

}

type ClientConfig struct {
	ServiceAccountID string
	KeyID string
	PrivateKey []byte
	FolderID string
}

type Client struct {
	name string
	folderID string
	client *yandex.SDK
	iamToken iamToken
	config ClientConfig
}

type YandexClouds struct {
	servers []Server
	clients []Client
}

type iamToken struct {
	IAMToken string  `json:"iamToken"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func NewYandexClouds(clientClouds []ClientConfig) (*YandexClouds) {
	y := YandexClouds{}
	for _, c := range clientClouds {
		token, err := getIAMToken(c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "yandex.NewYandexClouds: error getting IAM token for serviceAccountID: \"%s\" and KeyID: \"%s\", error: %v\n", c.ServiceAccountID, c.KeyID, err)
			continue
		}
		client, err := yandex.Build(context.Background(), yandex.Config{
			Credentials: yandex.NewIAMTokenCredentials(token.IAMToken),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "yandex.NewYandexClouds: error building yandex client for serviceAccountID: \"%s\" and KeyID: \"%s\", error: %v\n", c.ServiceAccountID, c.KeyID, err)
			continue
		}
		y.clients = append(y.clients, Client{
			name: c.ServiceAccountID,
			folderID: c.FolderID,
			client: client,
			iamToken: token,
			config: c,
		})
	}
	return &y
}

func (y *YandexClouds) getServers() {
	for _, c := range y.clients {

		// check expiration of IAM token
		if time.Now().After(c.iamToken.ExpiresAt) {
			fmt.Println("yandex.getServers: IAM token expired, getting new one")
			token, err := getIAMToken(c.config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "yandex.getServers: error getting IAM token for serviceAccountID: \"%s\" and KeyID: \"%s\", error: %v\n", c.config.ServiceAccountID, c.config.KeyID, err)
				continue
			}
			client, err := yandex.Build(context.Background(), yandex.Config{
				Credentials: yandex.NewIAMTokenCredentials(token.IAMToken),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "yandex.getServers: error building yandex client for serviceAccountID: \"%s\" and KeyID: \"%s\", error: %v\n", c.config.ServiceAccountID, c.config.KeyID, err)
				continue
			}
			c.client = client
			c.iamToken = token
		}

		
		servers, err := c.client.Compute().Instance().List(context.Background(), &compute.ListInstancesRequest{
			FolderId: c.folderID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "yandex.getServers: Error reading yandex servers: %v\n", err)
			continue
		}

		for _, s := range servers.GetInstances() {
			publicIP := ""
			privateIP := ""

			if len(s.NetworkInterfaces) > 0 {
				if s.NetworkInterfaces[0].PrimaryV4Address != nil && s.NetworkInterfaces[0].PrimaryV4Address.OneToOneNat != nil && s.NetworkInterfaces[0].PrimaryV4Address.OneToOneNat.Address != "" {
					publicIP = s.NetworkInterfaces[0].PrimaryV4Address.OneToOneNat.Address
				}
				if s.NetworkInterfaces[0].PrimaryV4Address != nil && s.NetworkInterfaces[0].PrimaryV4Address.Address != "" {
					privateIP = s.NetworkInterfaces[0].PrimaryV4Address.Address
				}
			}
			updateServerList.Lock()
			y.servers = append(y.servers, Server{
				ID: s.Id,
				Name: s.Name,
				Type: s.PlatformId,
				Zone: s.ZoneId,
				Region: s.ZoneId[:strings.LastIndex(s.ZoneId, "-")],
				PublicIP: net.ParseIP(publicIP),
				PrivateIP: net.ParseIP(privateIP),
				CpuCount: uint8(s.Resources.Cores),
				Memory: uint64(s.Resources.Memory),
			})
			updateServerList.Unlock()
		}
	}
}

func (y *YandexClouds) GetServers() []Server {
	y.getServers()
	return y.servers
}

func getIAMToken(config ClientConfig) (iamToken, error) {
	jot, err := signedToken(config)
	if err != nil {
		return iamToken{}, fmt.Errorf("yandex.getIAMToken: error signing token: %v", err)
	}

	resp, err := http.Post(
		audience,
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"jwt":"%s"}`, jot)),
	)
	defer resp.Body.Close()
	if err != nil {
		return iamToken{}, fmt.Errorf("yandex.getIAMToken: error getting IAM token: %v", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return iamToken{}, fmt.Errorf("yandex.getIAMToken: error getting IAM token: %s: %s", resp.Status, body)
	}

	var data iamToken
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return iamToken{}, fmt.Errorf("yandex.getIAMToken: error decoding IAM token: %v", err)
	}
	return data, nil
}

func signedToken(config ClientConfig) (string, error) {
	claims := jwt.RegisteredClaims{
			Issuer:    config.ServiceAccountID,
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			NotBefore: jwt.NewNumericDate(time.Now().UTC()),
			Audience:  []string{audience},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodPS256, claims)
	token.Header["kid"] = config.KeyID
  
	privateKey, err := loadPrivateKey(config.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("yandex.signedToken: error loading private key: %v", err)
	}
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("yandex.signedToken: error signing token: %v", err)
	}
	return signed, nil
}
  
func loadPrivateKey(key []byte) (*rsa.PrivateKey, error) {
	rsaPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM(key)
	if err != nil {
		return nil, fmt.Errorf("yandex.loadPrivateKey: error parsing private key: %v", err)
	}
	return rsaPrivateKey, nil
}
