package core

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tailwarden/komiser/models"
	"github.com/tailwarden/komiser/providers"
	oc "github.com/tailwarden/komiser/providers/k8s/opencost"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Nodes(ctx context.Context, client providers.ProviderClient) ([]models.Resource, error) {
	resources := make([]models.Resource, 0)

	var config metav1.ListOptions

	opencostEnabled := true
	nodesCost, err := oc.GetOpencostInfo(client.K8sClient.OpencostBaseUrl, "node")
	if err != nil {
		log.Errorf("ERROR: Couldn't get nodes info from OpenCost: %v", err)
		log.Warn("Opencost disabled")
		opencostEnabled = false
	}

	for {
		res, err := client.K8sClient.Client.CoreV1().Nodes().List(ctx, config)
		if err != nil {
			return nil, err
		}

		for _, node := range res.Items {
			tags := make([]models.Tag, 0)

			for key, value := range node.Labels {
				tags = append(tags, models.Tag{
					Key:   key,
					Value: value,
				})
			}

			cost := 0.0
			if opencostEnabled {
				cost = nodesCost[node.Name].TotalCost
			}

			resources = append(resources, models.Resource{
				Provider:   "Kubernetes",
				Account:    client.Name,
				Service:    "Node",
				ResourceId: string(node.UID),
				Name:       node.Name,
				Region:     node.Namespace,
				Cost:       cost,
				CreatedAt:  node.CreationTimestamp.Time,
				FetchedAt:  time.Now(),
				Tags:       tags,
			})
		}

		if res.GetContinue() == "" {
			break
		}

		config.Continue = res.GetContinue()
	}

	log.WithFields(log.Fields{
		"provider":  "Kubernetes",
		"account":   client.Name,
		"service":   "Node",
		"resources": len(resources),
	}).Info("Fetched resources")
	return resources, nil
}
