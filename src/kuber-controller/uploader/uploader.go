package uploader

import (
	"time"
	log "github.com/Sirupsen/logrus"
	"encoding/json"
	"net/http"
	"bytes"
	"kuber-controller/config"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const READ_SIZE uint32 = 50

type Payload struct {
	Key          string
	EventType    string
	Namespace    string
	ResourceType string
	Data         *interface{}
}

func UploadData(conf *config.Config) {

	subscriber := getSubscriber(conf)

	for true {
		conf.RingBuffer.PrintDetails()

		for true {
			data, size := conf.RingBuffer.ReadN(READ_SIZE)

			if size == 0 {
				log.Debug("There is no data to upload.")
				break
			}

			resp, err := SendData(data, subscriber)
			if err != nil {
				log.Warn("Error while sending data ", err)
				break
			}

			if resp != nil && resp.StatusCode == 200 {
				log.Info("Data is posted successfully")
				conf.RingBuffer.RemoveN(size)
				conf.RingBuffer.PrintDetails()
			} else {
				if resp != nil {
					log.Error("Data posting is failed with error ", resp.StatusCode)
				} else {
					log.Error("")
				}
				break
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func SendData(payload []*interface{}, subscriber *subscriber) (*http.Response, error) {
	jsonStr, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", subscriber.url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	setAuthHeaders(req, subscriber)
	client := &http.Client{}
	return client.Do(req)
}

func setAuthHeaders(r *http.Request, subscriber *subscriber)  {
	if subscriber.authType != "" {
		if subscriber.authType == "access-token" {
			r.Header.Set("Authorization", "Bearer " + subscriber.authCode)
		}
	}
}

type subscriber struct {
	url      string
	authType string
	authCode string
	cluster string

}

func getSubscriber(conf *config.Config) *subscriber {
	subscriber := &subscriber{}
	list, err := conf.Subscriberclient.ListSubscriber(meta_v1.ListOptions{})
	if err != nil {
		log.Error("Error while fetching subscribers list ", err)
		return nil
	} else {
		if list != nil && len(list.Items) > 0 {
			sub := list.Items[0]
			subscriber.url = sub.Spec.Url
			subscriber.authType = sub.Spec.AuthType
			subscriber.authCode = sub.Spec.AuthToken
			subscriber.cluster = sub.Spec.ClusterName
		} else {
			log.Info("There are no subscribers")
			return nil
		}
	}
	log.Info(*subscriber)
	return subscriber
}
