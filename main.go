package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
	"encoding/base64"

	"github.com/orangeAppsRu/custom-exporter/pkg/config"
	"github.com/orangeAppsRu/custom-exporter/pkg/filehash"
	"github.com/orangeAppsRu/custom-exporter/pkg/hetzner"
	"github.com/orangeAppsRu/custom-exporter/pkg/metrics"
	"github.com/orangeAppsRu/custom-exporter/pkg/network"
	"github.com/orangeAppsRu/custom-exporter/pkg/proc"
	"github.com/orangeAppsRu/custom-exporter/pkg/puppet"
	"github.com/orangeAppsRu/custom-exporter/pkg/system"
	"github.com/orangeAppsRu/custom-exporter/pkg/hetznercloud"
	"github.com/orangeAppsRu/custom-exporter/pkg/yandex"
	"github.com/orangeAppsRu/custom-exporter/pkg/aws"
	"github.com/orangeAppsRu/custom-exporter/pkg/util"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	version = "v0.0.17"
)

func main() {
	configFilePath := flag.String("config", "", "path to config file (env CONFIG by default)")
	versionFlag := flag.Bool("version", false, "print version")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

	if *configFilePath == "" {
		*configFilePath = os.Getenv("CONFIG")
	}

	if *configFilePath == "" {
		fmt.Println("Config file path is not provided. Use --config flag or set CONFIG environment variable.")
		return
	}

	cfg, err := config.ReadConfig(*configFilePath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	host := os.Getenv("HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8200"
	}
	listenAddr := fmt.Sprintf("%s:%s", host, port)

	metrics.RegistrMetrics(cfg)

	if cfg.FileHashCollector.Enabled {
		go func() {
			for {

				filesWithHash := []filehash.FileHash{}
				for _, filePath := range cfg.FileHashCollector.Files {
					number := 0.0
					if _, err := os.Stat(filePath); err == nil {
						number, err = filehash.Calculate(filePath)
						if err != nil {
							fmt.Printf("Error calculating hash for %s: %v\n", filePath, err)
							continue
						}
					}
					filesWithHash = append(filesWithHash, filehash.FileHash{
						File: filePath,
						Hash: number,
					})
				}
				metrics.UpdateFileHashMetrics(filesWithHash)
				time.Sleep(180 * time.Second)
			}
		}()
	}

	if cfg.PortCollector.Enabled {
		go func() {
			for {
				rTargets := network.CheckTargets(cfg.PortCollector.Targets)
				metrics.UpdateNetworkTargetsMetrics(rTargets)
				time.Sleep(60 * time.Second)
			}
		}()
	}

	if cfg.ProcessCollector.Enabled {
		go func() {
			procCollector := metrics.GetCustomProcCollector()
			for {
				if np, err := proc.CountProcesses(); err != nil {
					fmt.Printf("Error counting processes: %v\n", err)
				} else {
					metrics.UpdateProcessCountMetrics("all", np)
				}

				if npt, err := proc.CountProcessTypes(); err != nil {
					fmt.Printf("Error counting processes: %v\n", err)
				} else {
					for typeProcess, count := range npt {
						metrics.UpdateProcessCountMetrics(typeProcess, count)
					}
				}

				if processResources, err := proc.AggregateCPUTimeAndMemoryUsageByRegex(cfg.ProcessCollector.Processes); err != nil {
					fmt.Printf("Error aggregating process resources: %v\n", err)
				} else {
					for process, usage := range processResources {
						procCollector.Update("cpu_time", process, usage.CPUTime)
						metrics.UpdateProcessMemoryResidentMetrics(process, usage.ResidentMemory)
					}
				}

				if processRunningStatus, err := proc.FindProcessesByRegex(cfg.ProcessCollector.Processes); err != nil {
					fmt.Printf("Error finding processes: %v\n", err)
				} else {
					for process, count := range processRunningStatus {
						metrics.UpdateProcessRunningStatusMetrics(process, count)
					}
				}

				time.Sleep(15 * time.Second)
			}
		}()
	}

	if cfg.SystemCollector.Enabled {
		systemCollector := metrics.GetSystemCollector()
		go func() {
			for {
				// hostname checksum
				if hostnameChecksum, err := system.HostnameChecksum(); err != nil {
					fmt.Fprintf(os.Stderr, "Error getting hostname: %v\n", err)
				} else {
					metrics.UpdateHostnameChecksumMetrics(hostnameChecksum)
				}

				// uname checksum
				if unameChecksum, err := system.UnameChecksum(); err != nil {
					fmt.Fprintf(os.Stderr, "Error getting uname: %v\n", err)
				} else {
					metrics.UpdateUnameChecksumMetrics(unameChecksum)
				}

				// hostname
				if hostname, err := os.Hostname(); err != nil {
					fmt.Fprintf(os.Stderr, "Error getting hostname: %v\n", err)
				} else {
					metrics.UpdateHostnameMetrics(hostname)
				}

				// uptime
				if uptime, err := system.UptimeInSeconds(); err != nil {
					fmt.Fprintf(os.Stderr, "Error getting uptime: %v\n", err)
				} else {
					systemCollector.Update("uptime_seconds", uptime)
				}

				// count of login users
				if countLoginUsers, err := system.CountLoginUsers(); err != nil {
					fmt.Fprintf(os.Stderr, "Error getting count of login users: %v\n", err)
				} else {
					metrics.UpdateLoginUsersCountMetrics(countLoginUsers)
				}

				time.Sleep(60 * time.Second)
			}
		}()
	}

	if cfg.PuppetCollector.Enabled {
		go func() {
			for {
				p := puppet.NewPuppet(cfg.PuppetCollector.LastRunReportPath)

				metrics.UpdatePuppetCatalogLastCompileTimestampMetrics(p.CheckCatalogLastCompile())
				metrics.UpdatePuppetCatalogLastCompileStatusMetrics(p.CheckCatalogLastCompileStatus())

				time.Sleep(300 * time.Second)
			}
		}()
	}

	if cfg.HetznerCollector.Enabled {
		hrobotUser := os.Getenv("HROBOT_USER")
		if hrobotUser == "" {
			fmt.Fprintf(os.Stderr, "Error: env \"HROBOT_USER\" is required if hetznerCollector is enabled\n")
			os.Exit(1)
		}

		hrobotPass := os.Getenv("HROBOT_PASS")
		if hrobotPass == "" {
			fmt.Fprintf(os.Stderr, "Error: env \"HROBOT_PASS\" is required if hetznerCollector is enabled\n")
			os.Exit(1)
		}


		go func() {
			if cfg.HetznerCollector.RandomSleepBeforeStart {
				util.RandomSleep(1, 60, "Hetzner collector before start")
			}
			fmt.Println("Starting Hetzner collector")
			hetzner := hetzner.NewHetzner(hrobotUser, hrobotPass)
			for {
				metrics.UpdateHetznerServersMetrics(hetzner.GetServers())
				time.Sleep(600 * time.Second)
			}
		}()
	}

	if cfg.HetznerCloudCollector.Enabled {

		if os.Getenv("HCLOUD_TOKEN") == "" && os.Getenv("HCLOUD_TOKEN_0") == "" {
			fmt.Fprintf(os.Stderr, "Error: env \"HCLOUD_TOKEN\" or \"HCLOUD_TOKEN_<number>\" from 0 is required if hetznerCloudCollector is enabled\n")
			os.Exit(1)
		}

		var hcloudConfigs []hetznercloud.ClientConfig

		if os.Getenv("HCLOUD_TOKEN") != "" {
			hcloudToken := os.Getenv("HCLOUD_TOKEN")
			hcloudConfigs = append(hcloudConfigs, hetznercloud.ClientConfig{
				Token: hcloudToken,
			})
		} else {
			for i := 0; ; i++ {
				hcloudToken := os.Getenv(fmt.Sprintf("HCLOUD_TOKEN_%d", i))
				if hcloudToken == "" {
					break
				}
				hcloudConfigs = append(hcloudConfigs, hetznercloud.ClientConfig{
					Token: hcloudToken,
				})
			}
		}
		
		go func() {
			if cfg.HetznerCloudCollector.RandomSleepBeforeStart {
				util.RandomSleep(1, 60, "Hetzner Cloud collector before start")
			}
			fmt.Println("Starting Hetzner Cloud collector")
			
			hetznerClouds := hetznercloud.NewHetznerClouds(hcloudConfigs)
			for {
				metrics.UpdateHetznerCloudServersMetrics(hetznerClouds.GetServers())
				time.Sleep(600 * time.Second)
			}
		}()
	}

	if cfg.YandexCloudCollector.Enabled {

		if os.Getenv("YANDEX_CLOUD_SERVICE_ACCOUNT_ID") == "" && os.Getenv("YANDEX_CLOUD_SERVICE_ACCOUNT_ID_0") == "" {
			fmt.Fprintf(os.Stderr, "Error: env \"YANDEX_CLOUD_SERVICE_ACCOUNT_ID\" or \"YANDEX_CLOUD_SERVICE_ACCOUNT_ID<number>\" from 0 is required if yandexCloudCollector is enabled\n")
			os.Exit(1)
		}

		var yandexConfigs []yandex.ClientConfig

		if os.Getenv("YANDEX_CLOUD_SERVICE_ACCOUNT_ID") != "" {
			
			if os.Getenv("YANDEX_CLOUD_SERVICE_ACCOUNT_KEY_ID") == "" || os.Getenv("YANDEX_CLOUD_SERVICE_ACCOUNT_PRIVATE_KEY") == "" || os.Getenv("YANDEX_CLOUD_SERVICE_ACCOUNT_FOLDER_ID") == "" {
				fmt.Fprintf(os.Stderr, "Error: env \"YANDEX_CLOUD_SERVICE_ACCOUNT_KEY_ID\" and \"YANDEX_CLOUD_SERVICE_ACCOUNT_PRIVATE_KEY\" and \"YANDEX_CLOUD_SERVICE_ACCOUNT_FOLDER_ID\" are required if yandexCloudCollector is enabled\n")
				os.Exit(1)
			}

			yaServiceAccountId := os.Getenv("YANDEX_CLOUD_SERVICE_ACCOUNT_ID")
			yaKeyID := os.Getenv("YANDEX_CLOUD_SERVICE_ACCOUNT_KEY_ID")
			yaFolderId := os.Getenv("YANDEX_CLOUD_SERVICE_ACCOUNT_FOLDER_ID")
			yaPrivateKey, err := base64.StdEncoding.DecodeString(os.Getenv("YANDEX_CLOUD_SERVICE_ACCOUNT_PRIVATE_KEY"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: env \"YANDEX_CLOUD_SERVICE_ACCOUNT_PRIVATE_KEY\" is not base64 encoded\n")
				os.Exit(1)
			}

			yandexConfigs = append(yandexConfigs, yandex.ClientConfig{
				ServiceAccountID: yaServiceAccountId,
				KeyID: yaKeyID,
				PrivateKey: yaPrivateKey,
				FolderID: yaFolderId,
			})
		} else {
			for i := 0; ; i++ {
				yaServiceAccountId := os.Getenv(fmt.Sprintf("YANDEX_CLOUD_SERVICE_ACCOUNT_ID_%d", i))
				if yaServiceAccountId == "" {
					break
				}

				if os.Getenv(fmt.Sprintf("YANDEX_CLOUD_SERVICE_ACCOUNT_KEY_ID_%d", i)) == "" || os.Getenv(fmt.Sprintf("YANDEX_CLOUD_SERVICE_ACCOUNT_PRIVATE_KEY_%d", i)) == "" || os.Getenv(fmt.Sprintf("YANDEX_CLOUD_SERVICE_ACCOUNT_FOLDER_ID_%d", i)) == ""{
					fmt.Fprintf(os.Stderr, "Error: env \"YANDEX_CLOUD_SERVICE_ACCOUNT_KEY_ID_%d\" and \"YANDEX_CLOUD_SERVICE_ACCOUNT_PRIVATE_KEY_%d\" and \"YANDEX_CLOUD_SERVICE_ACCOUNT_FOLDER_ID_%d\" are required if YANDEX_CLOUD_SERVICE_ACCOUNT_ID_%d is exist and yandexCloudCollector is enabled\n", i, i, i, i)
					os.Exit(1)
				}

				yaKeyID := os.Getenv(fmt.Sprintf("YANDEX_CLOUD_SERVICE_ACCOUNT_KEY_ID_%d", i))
				yaFolderId := os.Getenv(fmt.Sprintf("YANDEX_CLOUD_SERVICE_ACCOUNT_FOLDER_ID_%d", i))
				yaPrivateKey, err := base64.StdEncoding.DecodeString(os.Getenv(fmt.Sprintf("YANDEX_CLOUD_SERVICE_ACCOUNT_PRIVATE_KEY_%d", i)))
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: env \"YANDEX_CLOUD_SERVICE_ACCOUNT_PRIVATE_KEY_%d\" is not base64 encoded\n", i)
					os.Exit(1)
				}
				
				yandexConfigs = append(yandexConfigs, yandex.ClientConfig{
					ServiceAccountID: yaServiceAccountId,
					KeyID: yaKeyID,
					PrivateKey: yaPrivateKey,
					FolderID: yaFolderId,
				})
			}
		}
		
		go func() {
			if cfg.YandexCloudCollector.RandomSleepBeforeStart {
				util.RandomSleep(1, 60, "Yandex Cloud collector before start")
			}
			fmt.Println("Starting Yandex Cloud collector")
			
			yandexClouds := yandex.NewYandexClouds(yandexConfigs)
			for {
				metrics.UpdateYandexCloudServersMetrics(yandexClouds.GetServers())
				time.Sleep(600 * time.Second)
			}
		}()
	}

	if cfg.AWSCloudCollector.Enabled {

		if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_ACCESS_KEY_ID_0") == "" {
			fmt.Fprintf(os.Stderr, "Error: env \"AWS_ACCESS_KEY_ID\" or \"AWS_ACCESS_KEY_ID_<number>\" from 0 is required if awsCloudCollector is enabled\n")
			os.Exit(1)
		}

		var awsConfigs []aws.ClientConfig

		if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
			
			if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" || os.Getenv("AWS_DEFAULT_REGION") == "" {
				fmt.Fprintf(os.Stderr, "Error: env \"AWS_ACCESS_KEY_ID\" and \"AWS_SECRET_ACCESS_KEY\" and \"AWS_DEFAULT_REGION\" are required if awsCloudCollector is enabled\n")
				os.Exit(1)
			}

			awsAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
			awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
			awsRegion := os.Getenv("AWS_DEFAULT_REGION")
			awsConfigs = append(awsConfigs, aws.ClientConfig{
				AccessKeyID: awsAccessKeyID,
				SecretAccessKey: awsSecretAccessKey,
				Region: awsRegion,
			})

		} else {
			for i := 0; ; i++ {
				awsAccessKeyID := os.Getenv(fmt.Sprintf("AWS_ACCESS_KEY_ID_%d", i))
				if awsAccessKeyID == "" {
					break
				}

				if os.Getenv(fmt.Sprintf("YANDEX_CLOUD_SERVICE_ACCOUNT_KEY_ID_%d", i)) == "" || os.Getenv(fmt.Sprintf("AWS_SECRET_ACCESS_KEY_%d", i)) == "" || os.Getenv(fmt.Sprintf("AWS_DEFAULT_REGION_%d", i)) == ""{
					fmt.Fprintf(os.Stderr, "Error: env \"AWS_ACCESS_KEY_ID_%d\" and \"AWS_SECRET_ACCESS_KEY_%d\" and \"AWS_DEFAULT_REGION_%d\" are required if AWS_SECRET_ACCESS_KEY_%d is exist and awsCloudCollector is enabled\n", i, i, i, i)
					os.Exit(1)
				}

				awsSecretAccessKey := os.Getenv(fmt.Sprintf("YANDEX_CLOUD_SERVICE_ACCOUNT_KEY_ID_%d", i))
				awsRegion := os.Getenv(fmt.Sprintf("YANDEX_CLOUD_SERVICE_ACCOUNT_FOLDER_ID_%d", i))
				awsConfigs = append(awsConfigs, aws.ClientConfig{
					AccessKeyID: awsAccessKeyID,
					SecretAccessKey: awsSecretAccessKey,
					Region: awsRegion,
				})
			}
		}

		go func() {
			if cfg.AWSCloudCollector.RandomSleepBeforeStart {
				util.RandomSleep(1, 60, "AWS Cloud collector before start")
			}
			fmt.Println("Starting AWS Cloud collector")
			
			awsClouds := aws.NewAwsClouds(awsConfigs)
			for {
				metrics.UpdateAWSCloudServersMetrics(awsClouds.GetServers())
				time.Sleep(600 * time.Second)
			}
		}()

	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})


	fmt.Println("Prometheus metrics server started at", listenAddr)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
