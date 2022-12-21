package app

import (
	"context"
	"externaldns-adguard-go/internal/adguard"
	"externaldns-adguard-go/internal/config"
	"externaldns-adguard-go/internal/database"
	"fmt"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path/filepath"
)

type dnsEvent struct {
	Type        watch.EventType
	Entry       database.DnsEntry
	HasHostname bool
}

func syncAdGuardWithDatabase(client *adguard.AdGuardClient, db *database.DnsDatabase) {
	log.Println("Syncing database to AdGuard")
	var dbEntries []database.DnsEntry
	db.GetAll(&dbEntries)
	agEntries := make(map[string]string)
	client.GetEntries(&agEntries)
	for _, dbEntry := range dbEntries {
		ip, exists := agEntries[dbEntry.Hostname]
		if exists && ip != dbEntry.IP {
			log.Printf("Updating adguard record: %s from %s\n", dbEntry, ip)
			err := client.DeleteEntry(dbEntry.Hostname, ip)
			if err != nil {
				log.Fatalf("could not delete adguard entry: %v", err)
			}
			err = client.CreateEntry(dbEntry.Hostname, dbEntry.IP)
			if err != nil {
				log.Fatalf("could not create adguard entry: %v", err)
			}
		} else if !exists {
			log.Printf("Adding adguard record: %s\n", dbEntry)
			err := client.CreateEntry(dbEntry.Hostname, dbEntry.IP)
			if err != nil {
				log.Fatalf("could not create adguard entry: %v", err)
			}

		}
	}
}

func receiveEvent(appConfig config.Config, output *dnsEvent, serviceEventChan <-chan watch.Event, ingressEventChan <-chan watch.Event) error {
	var hostname string
	select {
	case event := <-serviceEventChan:
		output.Type = event.Type
		entity, ok := event.Object.(*coreV1.Service)
		if !ok {
			return fmt.Errorf("unexpected event received from Service watcher: %v", event)
		}
		hostname, output.HasHostname = entity.Annotations[appConfig.Annotation]
		output.Entry = database.DnsEntry{
			EntryType: "service",
			Namespace: entity.Namespace,
			Name:      entity.Name,
			Hostname:  hostname,
		}
		if len(entity.Status.LoadBalancer.Ingress) > 0 {
			output.Entry.IP = entity.Status.LoadBalancer.Ingress[0].IP
		}
	case event := <-ingressEventChan:
		output.Type = event.Type
		entity, ok := event.Object.(*networkingV1.Ingress)
		if !ok {
			return fmt.Errorf("unexpected event received from Ingress watcher: %v", event)
		}
		hostname, output.HasHostname = entity.Annotations[appConfig.Annotation]
		output.Entry = database.DnsEntry{
			EntryType: "ingress",
			Namespace: entity.Namespace,
			Name:      entity.Name,
			Hostname:  hostname,
		}
		if len(entity.Status.LoadBalancer.Ingress) > 0 {
			output.Entry.IP = entity.Status.LoadBalancer.Ingress[0].IP
		}
	}
	return nil
}

func createKubernetesClient(config config.Config) (*kubernetes.Clientset, error) {
	var kubeconfigPath string
	if config.Mode == "DEV" {
		kubeconfigPath = filepath.Join(
			os.Getenv("HOME"), ".kube", "config",
		)
	}
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("could not build kubernetes config from path '%s', %v", kubeconfigPath, err)
	}
	k8s, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("could not initialitze kubernetes client %v", err)
	}
	return k8s, nil
}

func RunApp() {
	appConfig, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("can't load environment app.env: %v", err)
	}
	//log.Println("Current config:", appConfig)
	k8s, err := createKubernetesClient(appConfig)
	if err != nil {
		log.Fatal(err)
	}

	// initialize database and adguard client
	db := database.NewDnsDatabase(appConfig.DatabaseFile)
	dnsClient := adguard.NewAdGuardClient(
		appConfig.AdGuardScheme, appConfig.AdGuardUrl,
		appConfig.AdGuardUsername, appConfig.AdGuardPassword, appConfig.AdGuardLogging)
	err = dnsClient.RefreshEntries()
	if err != nil {
		log.Fatalf("could not get adguard cache: %v", err)
	}
	syncAdGuardWithDatabase(dnsClient, db)

	for {
		// watch all services
		coreV1api := k8s.CoreV1()
		serviceWatcher, err := coreV1api.Services("").Watch(context.TODO(), metaV1.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}
		serviceEventChan := serviceWatcher.ResultChan()

		// watch all ingresses
		networkingV1api := k8s.NetworkingV1()
		ingressWatcher, err := networkingV1api.Ingresses("").Watch(context.TODO(), metaV1.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}
		ingressEventChan := ingressWatcher.ResultChan()

		log.Println("Listening for events")
		for {
			event := dnsEvent{}
			err := receiveEvent(appConfig, &event, serviceEventChan, ingressEventChan)
			if err != nil {
				log.Printf("Error receiving event: %v", err)
				log.Println("Restarting event loop")
				break
			}
			var existingEntry database.DnsEntry
			hasEntry := db.GetByName(&existingEntry, event.Entry.EntryType, event.Entry.Namespace, event.Entry.Name)
			if event.Type == watch.Added || event.Type == watch.Modified {
				if event.HasHostname {
					if event.Entry.IP == "" {
						log.Printf("Ignoring Entry %s\n", event.Entry)
						continue
					}
					if hasEntry {
						if existingEntry.Hostname != event.Entry.Hostname {
							log.Printf("Updating Entry %s to %s\n", existingEntry, event.Entry)
							err = dnsClient.DeleteEntry(existingEntry.Hostname, existingEntry.IP)
							if err != nil {
								log.Fatalf("could not delete adguard entry: %v", err)
							}
							err = dnsClient.CreateEntry(event.Entry.Hostname, event.Entry.IP)
							if err != nil {
								log.Fatalf("could not create adguard entry: %v", err)
							}
							existingEntry.Hostname = event.Entry.Hostname
							db.UpdateEntry(&existingEntry)
						}
					} else {
						log.Printf("Adding Entry %s\n", event.Entry)
						ipInAdguard, adguardEntryExists := dnsClient.GetEntry(event.Entry.Hostname)
						if adguardEntryExists && ipInAdguard != event.Entry.IP {
							err = dnsClient.DeleteEntry(event.Entry.Hostname, ipInAdguard)
							if err != nil {
								log.Fatalf("could not delete adguard entry: %v", err)
							}

						} else if !adguardEntryExists {
							err = dnsClient.CreateEntry(event.Entry.Hostname, event.Entry.IP)
							if err != nil {
								log.Fatalf("could not create adguard entry: %v", err)
							}

						}
						db.AddEntry(&event.Entry)
					}
				} else if hasEntry {
					log.Printf("Deleting stale Entry %s\n", existingEntry)
					db.DeleteEntry(&existingEntry)
					ipInAdguard, adguardEntryExists := dnsClient.GetEntry(existingEntry.Hostname)
					if adguardEntryExists && ipInAdguard == event.Entry.IP {
						err = dnsClient.DeleteEntry(existingEntry.Hostname, existingEntry.IP)
						if err != nil {
							log.Fatalf("could not delete adguard entry: %v", err)
						}
					}
				}
			} else if event.Type == watch.Deleted {
				if hasEntry {
					log.Printf("Deleting Entry %s\n", existingEntry)
					db.DeleteEntry(&existingEntry)
					err = dnsClient.DeleteEntry(existingEntry.Hostname, existingEntry.IP)
					if err != nil {
						log.Fatalf("could not delete adguard entry: %v", err)
					}

				}
			}
		}
	}
}
