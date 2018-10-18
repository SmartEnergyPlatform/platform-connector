# SEPL-Connector

This service provides a communication endpoint for gateways to 
* send device-metadata
* send device-events / sensor data
* receive commands for devices

The communication with gateways occurs by websocket.
The connector logs connects/disconnects, 
formats event-data to the platform-intern format  
and relays them to kafka.

## Connector-Handlers
* [clear](#clear)
* [put](#put)
* [remove](#remove)
* [delete](#delete)
* [commit](#commit)
* [event](#event)
* [command-response](#command-response)

## Envelope

### Client-To-Server
```
{"handler":"<<handler_name>>","token":"<<token>>","payload":<<payload>>}
```

### Server-To-Client
```
{"status":<<status_code>>,"handler":"<<handler_name>>","token":"<<token>>","content_type":"<<content_type>>","payload":<<payload>>}
```

### Fields
 * handler:
    * contains called handler
    * for client-to-server it can contain the strings _clear_, _put_, _remove_, _commit_, _event_, _response_
    * for server-to-client it can contain the strings _response_ and _command_
 * token:
    * used to link a response to a request
    * user defines the token in request envelope
    * platform returns same token in matching response envelope
 * payload:
    * contains information to transmit
    * can be json object, array, number or string
 * content_type:
    * only in platform response or command
    * hint for type of payload
    * can be _int64_, _map_, _slice_, _string_
    * from golang `reflect.TypeOf(ct).Kind().String()` --> type.go kindNames
 * status:
    * informs about success or failure of request
    * only in platform response
    * semantics of http status codes
    * currently only 200, 400 and 500 in use


## Client-Docu
    
### Connecting
 * WS to `wss://<<address>>:<<config.WssPort>>/` or `ws://<<address>>:<<config.WsPort>>/`  
 * immediately after connecting Client has to send his Credentials: 
   `{user: "<<user_name>>", pw: <<user_password>>, token: "<<token>>", gid: "<<gateway_id>>"}`
 * credentials request does not use the envelope
 * response arrives on the normal _response_ handler
 * response-payload: `{"gid": "<<gateway_id>>", "hash": "<<hash>>"}`
 * if the requested gateway_id is unknown the platform will create a new gateway and returns its id
 * the returned _hash_ is the same as from the last successful _commit_ request 
 
#### Procedure

 1. handshake with previously stored gateway_id or "" as gid
 2. save returned gid
 3. compute hash of current devices and compare with response hash
 4. if hashes are not equal
    1. use _clear_ handler to reset gateway
    2. use _put_ handler to add devices to gateway
    3. use _commit_ handler to commit new hash to gateway
 
### Handler-Server-Side
 
#### Clear
 * resets gateway to initial state, without hash and devices, should be called if received hash is not consistent to local devices hash
 * handler: _clear_
 * payload: empty
 * response:
    * status: 200, payload: "ok", content_type: "string"
    * status: 500, payload: "error_desc", content_type: "string"
     
```
{  
   "handler":"clear",
   "token":"dca714ed-9ebf-47fb-84a1-86ac8f16009f",
   "payload":null
}
```

#### Put
 * start listening to commands for this device; allows events for this device to be send; if the device is unknown, it will be created; changes of device name and tags will be transmitted to the iot-repository
 * handler: _put_
 * payload: `{"iot_type": "<<device_type_id>>", "uri": "<<device_uri>>", "name": "<<device_name>>", "tags": ["<<tag_1>>"]}`
 * response:
    * status: 200, payload: "ok", content_type: "string"
    * status: 400, payload: "error_desc", content_type: "string"
    * status: 500, payload: "error_desc", content_type: "string"
 * payload properties:
    * iot_type: id of deviceType from iot-repository
    * uri: identifier for the device; should be unique
    * tags: a list of tags; a tag is a string with 2 parts separated by ':'; only the second part will be displayed in the ui; the first part is used as a key to identify tag changes; `["<<key>>:<<value>>", "location:leipzig"]`  
    
```
{  
   "handler":"put",
   "token":"3b5ff3b1-fd16-45cb-9f09-8650346c544e",
   "payload":{  
      "name":"Dummy Meter",
      "tags":[  
         "type:Smart Meter"
      ],
      "uri":"3iguegr9hgh9",
      "iot_type":"iot#40c41048-3910-468f-ae74-7425dde90963"
   }
}
```
    
#### Disconnect
 * stop listening to commands for this device; events for this device will return error
 * handler: _disconnect_
 * payload: `"<<device_uri>>"`
 * response:
    * status: 200, payload: "ok", content_type: "string"
    * status: 400, payload: "error_desc", content_type: "string"
    * status: 500, payload: "error_desc", content_type: "string"
    
```
{  
   "handler":"delete",
   "token":"586bab4e-2c63-429e-9561-1fb61e652199",
   "payload":"3iguegr9hgh9"
}
```

#### Delete
 * deletes device from the SEPL-Platform and stops listening to commands for this device
 * handler: _delete_
 * payload: `"<<device_uri>>"`
 * response:
    * status: 200, payload: "ok", content_type: "string"
    * status: 400, payload: "error_desc", content_type: "string"
    * status: 500, payload: "error_desc", content_type: "string"
    
```
{  
   "handler":"remove",
   "token":"586bab4e-2c63-429e-9561-1fb61e652199",
   "payload":"3iguegr9hgh9"
}
```
    
#### Commit
 * associate all currently listened to devices with the gateway; on reconnect all the current listening state will be restored; saves hash for the gateway; hash will be used in connection handshake, to check for device changes
 * handler: _commit_
 * payload: `"<<hash>>"`
 * response:
     * status: 200, payload: "ok", content_type: "string"
     * status: 400, payload: "error_desc", content_type: "string"
     * status: 500, payload: "error_desc", content_type: "string"
    
```
{  
   "handler":"commit",
   "token":"f8f7a023-cf9c-4a64-b64a-27d45aed7180",
   "payload":"1fa01e4811b385269335cc10299c06d8d4b7a632"
}
```
 
#### Event
 * send data to the platform
 * handler: _event_
 * payload: `{"device_uri": "<<device_uri>>", "service_uri": "<<service_uri>>", "value": <<protocol_parts>> }`
    * payload.value (<<protocol_parts>>) is a list of protocol_parts: 
     `[{"name": "<<protocol_part_name>>", "value": "<<protocol_part_value>>"}]`
    * protocol_part_name is the name of a protocolpart (e.g. _header_ or _body_ in the http protocol)
    * protocol_part_value is the value associated with the protocolpart
    * protocol_part_value is always transmitted as a string
 * response:
    * status: 200, payload: "ok", content_type: "string"
    * status: 400, payload: "error_desc", content_type: "string"
    * status: 500, payload: "error_desc", content_type: "string"
    
```
{  
   "handler":"event",
   "token":"52e3e326-d53e-4997-b739-f404b749eb78",
   "payload":{  
      "device_uri":"c98b2c1a-ba68",
      "service_uri":"sepl_get",
      "value":[  
         {  
            "name":"body"
            "value":"{\"value\": 93.786, \"unit\": \"kWh\", \"time\": \"2017-10-16T11:36:18.306126\"}",
         }
      ]
   }
}
```

#### Command-Response
 * send response to received _command_
 * handler: _response_
 * payload: reuse command message and overwrite payload.protocol_parts with the command results
    * protocol_parts are as described in events
    * protocol_parts can be null if no result output is expected
    * for the user relevant fields are device_url, service_url, and protocol_parts
    * other fields are mandatory and should remain unchanged but are irrelevant for the user (these fields may be removed in the future)
 * response:
    * status: 200, payload: "ok", content_type: "string"
    * status: 400, payload: "error_desc", content_type: "string"

```
{  
   "handler":"response",
   "token":"c4b7e15a-1460-4fcf-87a4-91fed6d849d4",
   "payload":{  
      "device_url":"sdafgihsidfaggdfiuh",
      "service_url":"on",
      "protocol_parts":[  
         {  
            "name":"body",
            "value":"200"
         }
      ],
      ...
   }
}

```
 

### Handler-Client-Side

#### Response
 * respond to user request
 * handler: _response_
 * ref [envelope](#envelope)
 
```
{  
   "status":200,
   "handler":"response",
   "token":"8da3a65f-ba3c-45bb-a49d-6b060472ea52",
   "content_type":"string",
   "payload":"ok"
}
```

#### Command
 * receive command for device
 * respond with [command-response](#command-response)
 * payload as described in command-response
 
```
{  
   "handler":"command",
   "content_type":"map",
   "payload":{  
      "device_url":"sdafgihsidfaggdfiuh",
      "service_url":"on",
      "protocol_parts":[  
         {  
            "name":"body",
            "value":"{\n    \"on\": true\n}\n"
         }
      ],
      ...
   }
}
```










## Server-Side-Docu

### Commands and Command-Responses
* Command-Responses are forwarded to the config.KafkaResponseTopic kafka topic
* Commands are read from the config.KafkaConsumerTopic kafka topic
* From Kafka received commands have the following format: `<<device_id>>:<<command_msg>>`
* device_id is the id of the device from the SEPL-Device-Repository, where all '#' are replaced with '_'
* the 'device_id' prefix is used for the message forwarding in ktrouter
* ktrouter (fg-seits/ktrouter) forwards commands from config.KafkaSourceTopic to config.KafkaConsumerTopic if the command corresponds to a device registered by [listen_to_devices](#listen-to-devices)
* the command is forwarded to the websocket without the 'device_id:' prefix (but with the command prefix)

**Kafka-Command-Example:**
```
iot_e17ac518-6068-4f0c-ad65-88796b020705:{  
   "worker_id":"3de19f29-1d4f-483e-b4aa-4147e1c0db40",
   "task_id":"59bd2465-6d48-11e7-9b84-02f1c81cd8d8",
   "device_url":"ZWAY_unique_controller_id_DummyDevice_10",
   "service_url":"sepl_get",
   "protocol_parts":null,
   "device_instance_id":"iot#e17ac518-6068-4f0c-ad65-88796b020705",
   "service_name":"sepl_get",
   "output_name":"Task_1kmeuz3",
   "time":"1500554317"
}
```

**Kafka-Command-Response-Example:**
```
{  
   "worker_id":"3de19f29-1d4f-483e-b4aa-4147e1c0db40",
   "task_id":"59bd2465-6d48-11e7-9b84-02f1c81cd8d8",
   "device_url":"ZWAY_unique_controller_id_DummyDevice_10",
   "service_url":"sepl_get",
   "protocol_parts":[  
      {  
         "name":"metrics",
         "value":"{\"level\":\"on\",\"title\":\"Dummy Device\",\"updateTime\":1500553405}"
      }
   ],
   "device_instance_id":"iot#e17ac518-6068-4f0c-ad65-88796b020705",
   "service_name":"sepl_get",
   "output_name":"Task_1kmeuz3",
   "time":"1500554317"
}
```

### Events
* Events are transformed from the in SEPL-Device-Repository specified format to json and forwarded to kafka 
    * with a wrapper to a service specific topic (SEPL service.id with '_' replacing '#')
    * with a wrapper and prefix to config.KafkaEventTopic (SEPL device.id + "." + service.id with '_' replacing '#')
* if a eventfilter for the service is registered and running the config.KafkaEventTopic message will be used and forwarded by the ktrouter using the given prefix.
* other analytics services can use the service specific topic

#### Example
**Client-Event**
```
event.7cd2a0f9-3026-f548-f5b4-499707b9509e:{  
   "device_uri":"ZWAY_unique_controller_id_DummyDevice_10",
   "service_uri":"sepl_get",
   "value":[  
      {  
         "name":"metrics",
         "value":"{\"level\":\"on\",\"title\":\"Dummy Device\",\"updateTime\":1500550320}"
      }
   ]
}
```

**Kafka-Message to config.KafkaEventTopic**
```
iot_e17ac518-6068-4f0c-ad65-88796b020705.iot_5c290b9e-fa92-4a1a-a294-7303188e0076:{  
    "device_id":"iot#e17ac518-6068-4f0c-ad65-88796b020705",
    "service_id":"iot#5c290b9e-fa92-4a1a-a294-7303188e0076",
    "value":{  
        "metrics":{  
            "level":"on",
            "title":"Dummy Device",
            "updateTime":1500550320
        }
    }
}
```

**Kafka-Message to service.id (iot_5c290b9e-fa92-4a1a-a294-7303188e0076)**
```
{  
    "device_id":"iot#e17ac518-6068-4f0c-ad65-88796b020705",
    "service_id":"iot#5c290b9e-fa92-4a1a-a294-7303188e0076",
    "value":{  
        "metrics":{  
            "level":"on",
            "title":"Dummy Device",
            "updateTime":1500550320
        }
    }
}
```

**Note**
that the field *metrics* in the Kafka-Messages has not the same meaning as the *metrics* in the Client-Message. 
In the Client-Message it is the name of the Protocol-Part.
In the Kafka-Message it is the User-Given name of the output-variable.
That these names are equal is coincidental because the user chose to use the same name for this variable as the for the protocol-part. 


