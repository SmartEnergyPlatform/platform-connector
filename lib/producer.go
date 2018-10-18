/*
 * Copyright 2018 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lib

import (
	"github.com/SmartEnergyPlatform/platform-connector/util"
	"log"
	"time"

	"sync"

	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kazoo-go"
)

var onceProducer sync.Once
var producer sarama.AsyncProducer

func InitProducer() sarama.AsyncProducer {
	var kz *kazoo.Kazoo
	kz, err := kazoo.NewKazooFromConnectionString(util.Config.ZookeeperUrl, nil)
	if err != nil {
		log.Fatal("error in kazoo.NewKazooFromConnectionString()", err)
	}
	broker, err := kz.BrokerList()
	kz.Close()

	if err != nil {
		log.Fatal("error in kz.BrokerList()", err)
	}

	sarama_conf := sarama.NewConfig()
	sarama_conf.Version = sarama.V0_10_0_1
	producer, err = sarama.NewAsyncProducer(broker, sarama_conf)
	if err != nil {
		log.Fatal("error in sarama.NewAsyncProducer()", broker, err)
	}
	return producer
}

func Produce(topic string, message string) {
	onceProducer.Do(func() {
		producer = InitProducer()
	})
	if message != "topic_init" {
		log.Println("produce kafka msg: ", topic, message)
	}
	producer.Input() <- &sarama.ProducerMessage{Topic: topic, Key: nil, Value: sarama.StringEncoder(message), Timestamp: time.Now()}
}

func CloseProducer() {
	onceProducer.Do(func() {
		producer = InitProducer()
	})
	producer.Close()
}
