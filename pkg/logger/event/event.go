/*
 * Copyright 2018 AccelByte Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package event

import (
	"github.com/AccelByte/public-source-ip"
	"github.com/emicklei/go-restful"
	"github.com/fatih/structs"
	"github.com/sirupsen/logrus"
)

const (
	eventLogAttribute = "EventLog"
	eventType         = "event"
)

type event struct {
	ID               int                    `structs:"event_id"`
	UserID           string                 `structs:"user_id"`
	Namespace        string                 `structs:"namespace"`
	ClientID         string                 `structs:"client_id"`
	TargetUserID     string                 `structs:"target_user_id"`
	TargetNamespace  string                 `structs:"target_namespace"`
	Realm            string                 `structs:"realm"`
	SourceIP         string                 `structs:"source_ip"`
	LogType          string                 `structs:"log_type"`
	level            logrus.Level           `structs:"-"`
	message          []interface{}          `structs:"-"`
	additionalFields map[string]interface{} `structs:"-"`
}

// ExtractAttribute is a function to extract userID, clientID and namespace from restful.Request
type ExtractAttribute func(req *restful.Request) (userID string, clientID string, namespace string)

// ExtractNull is null function for extracting attribute
var ExtractNull ExtractAttribute = func(_ *restful.Request) (userID string, clientID string, namespace string) {
	return "", "", ""
}

// Log is a filter that will log incoming request with AccelByte's Event Log Format
func Log(realm string, fn ExtractAttribute) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		evt := &event{
			LogType:  eventType,
			Realm:    realm,
			SourceIP: public_source_ip.PublicIP(req.Request),
			level:    logrus.InfoLevel,
		}
		req.SetAttribute(eventLogAttribute, evt)

		chain.ProcessFilter(req, resp)

		if evt.ID == 0 {
			return
		}

		evt.UserID, evt.ClientID, evt.Namespace = fn(req)

		fields := structs.Map(req.Attribute(eventLogAttribute))
		for key, value := range evt.additionalFields {
			fields[key] = value
		}

		log := logrus.WithFields(logrus.Fields(fields))

		switch evt.level {
		case logrus.FatalLevel:
			log.Fatal(evt.message...)
		case logrus.ErrorLevel:
			log.Error(evt.message...)
		case logrus.WarnLevel:
			log.Warn(evt.message...)
		case logrus.DebugLevel:
			log.Debug(evt.message...)
		default:
			log.Info(evt.message...)
		}
	}
}

// TargetUser injects the target user ID and namespace to current event in the request
func TargetUser(req *restful.Request, id, namespace string) {
	if evt := getEvent(req); evt != nil {
		evt.TargetUserID = id
		evt.TargetNamespace = namespace
	}
}

// AdditionalFields injects additional log fields to the event in the request
func AdditionalFields(req *restful.Request, fields map[string]interface{}) {
	if evt := getEvent(req); evt != nil {
		evt.additionalFields = fields
	}
}

// Debug sets the event level to debug along with the event ID and message
func Debug(req *restful.Request, eventID int, message ...interface{}) {
	if evt := getEvent(req); evt != nil {
		evt.ID = eventID
		evt.message = message
		evt.level = logrus.DebugLevel
	}
}

// Info sets the event level to info along with the event ID and message
func Info(req *restful.Request, eventID int, message ...interface{}) {
	if evt := getEvent(req); evt != nil {
		evt.ID = eventID
		evt.message = message
		evt.level = logrus.InfoLevel
	}
}

// Warn sets the event level to warn along with the event ID and message
func Warn(req *restful.Request, eventID int, message ...interface{}) {
	if evt := getEvent(req); evt != nil {
		evt.ID = eventID
		evt.message = message
		evt.level = logrus.WarnLevel
	}
}

// Error sets the event level to error along with the event ID and message
func Error(req *restful.Request, eventID int, message ...interface{}) {
	if evt := getEvent(req); evt != nil {
		evt.ID = eventID
		evt.message = message
		evt.level = logrus.ErrorLevel
	}
}

// Fatal sets the event level to fatal along with the event ID and message
func Fatal(req *restful.Request, eventID int, message ...interface{}) {
	if evt := getEvent(req); evt != nil {
		evt.ID = eventID
		evt.message = message
		evt.level = logrus.FatalLevel
	}
}

func getEvent(req *restful.Request) *event {
	evt, _ := req.Attribute(eventLogAttribute).(*event)
	return evt
}
