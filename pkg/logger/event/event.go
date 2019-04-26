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
	"github.com/emicklei/go-restful"
	"github.com/fatih/structs"
	"github.com/sirupsen/logrus"
)

const (
	eventLogAttribute     = "EventLog"
	millisecondTimeFormat = "2006-01-02T15:04:05.999Z07:00"
	logType               = "event"
	TraceIDKey            = "X-Ab-TraceID"
	SessionIDKey          = "X-Ab-SessionID"
)

type event struct {
	ID               int                    `structs:"event_id"`
	Type             int                    `structs:"event_type"`
	EventLevel       int                    `structs:"event_level"`
	Resp             int                    `structs:"resp"`
	Service          string                 `structs:"service"`
	ClientIDs        []string               `structs:"client_ids"`
	UserID           string                 `structs:"user_id"`
	TargetUserIDs    []string               `structs:"target_user_ids"`
	Namespace        string                 `structs:"namespace"`
	TargetNamespace  string                 `structs:"target_namespace"`
	TraceID          string                 `structs:"trace_id"`
	SessionID        string                 `structs:"session_id"`
	additionalFields map[string]interface{} `structs:"-"`
	Realm            string                 `structs:"realm"`
	topic            string                 `structs:"-"`
	Action           string                 `structs:"action"`
	Status           string                 `structs:"status"`
	LogType          string                 `structs:"log_type"`
	Message          string                 `structs:"msg"`
	level            logrus.Level           `structs:"-"`
}

// ExtractAttribute is a function to extract userID, clientID and namespace from restful.Request
type extractAttribute func(req *restful.Request) (userID string, clientID []string, namespace string, traceID string,
	sessionID string)

// extractNull is null function for extracting attribute
var extractNull extractAttribute = func(req *restful.Request) (userID string, clientID []string, namespace string,
	traceID string, sessionID string) {
	return "", []string{}, "", req.HeaderParameter(TraceIDKey), req.HeaderParameter(SessionIDKey)
}

func init() {
	logrus.SetFormatter(UTCFormatter{&logrus.TextFormatter{TimestampFormat: millisecondTimeFormat}})
}

// UTCFormatter implements logrus Formatter for custom log format
type UTCFormatter struct {
	logrus.Formatter
}

// Format implements logrus Format for forcing time to UTC
func (formatter UTCFormatter) Format(log *logrus.Entry) ([]byte, error) {
	log.Time = log.Time.UTC()
	return formatter.Formatter.Format(log)
}

// Log is a filter that will log incoming request with AccelByte's Event Log Format
func Log(realm string, service string, fn extractAttribute) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		evt := &event{
			Realm:   realm,
			Service: service,
			LogType: logType,
		}
		req.SetAttribute(eventLogAttribute, evt)

		chain.ProcessFilter(req, resp)

		if evt.ID == 0 {
			return
		}

		evt.UserID, evt.ClientIDs, evt.Namespace, evt.TraceID, evt.SessionID = fn(req)

		fields := structs.Map(req.Attribute(eventLogAttribute))
		for key, value := range evt.additionalFields {
			fields[key] = value
		}

		log := logrus.WithFields(logrus.Fields(fields))

		switch evt.level {
		case logrus.FatalLevel:
			log.Fatal(evt.Message)
		case logrus.ErrorLevel:
			log.Error(evt.Message)
		case logrus.WarnLevel:
			log.Warn(evt.Message)
		case logrus.DebugLevel:
			log.Debug(evt.Message)
		default:
			log.Info()
		}
	}
}

// TargetUser injects the target user ID and namespace to current event in the request
func TargetUser(req *restful.Request, id, namespace string) {
	if evt := getEvent(req); evt != nil {
		evt.TargetUserIDs = []string{id}
		evt.TargetNamespace = namespace
	}
}

// AdditionalFields injects additional log fields to the event in the request
func AdditionalFields(req *restful.Request, fields map[string]interface{}) {
	if evt := getEvent(req); evt != nil {
		evt.additionalFields = fields
	}
}

// Topic inject the topic to current event in the request
func Topic(req *restful.Request, topic string) {
	if evt := getEvent(req); evt != nil {
		evt.topic = topic
	}
}

// Action inject the topic to current event in the request
func Action(req *restful.Request, action string) {
	if evt := getEvent(req); evt != nil {
		evt.Action = action
	}
}

// Debug sets the event level to debug along with the event ID and message
func Debug(req *restful.Request, eventID int, eventType int, eventLevel int, msg string) {
	if evt := getEvent(req); evt != nil {
		evt.ID = eventID
		evt.Type = eventType
		evt.EventLevel = eventLevel
		evt.level = logrus.DebugLevel
		evt.Message = msg
	}
}

// Info sets the event level to info along with the event ID and message
func Info(req *restful.Request, eventID int, eventType int, eventLevel int, msg string) {
	if evt := getEvent(req); evt != nil {
		evt.ID = eventID
		evt.Type = eventType
		evt.EventLevel = eventLevel
		evt.level = logrus.InfoLevel
		evt.Message = msg
	}
}

// Warn sets the event level to warn along with the event ID and message
func Warn(req *restful.Request, eventID int, eventType int, eventLevel int, msg string) {
	if evt := getEvent(req); evt != nil {
		evt.ID = eventID
		evt.Type = eventType
		evt.EventLevel = eventLevel
		evt.level = logrus.WarnLevel
		evt.Message = msg
	}
}

// Error sets the event level to error along with the event ID and message
func Error(req *restful.Request, eventID int, eventType int, eventLevel int, msg string) {
	if evt := getEvent(req); evt != nil {
		evt.ID = eventID
		evt.Type = eventType
		evt.EventLevel = eventLevel
		evt.level = logrus.ErrorLevel
		evt.Message = msg
	}
}

// Fatal sets the event level to fatal along with the event ID and message
func Fatal(req *restful.Request, eventID int, eventType int, eventLevel int, msg string) {
	if evt := getEvent(req); evt != nil {
		evt.ID = eventID
		evt.Type = eventType
		evt.EventLevel = eventLevel
		evt.level = logrus.FatalLevel
		evt.Message = msg
	}
}

func getEvent(req *restful.Request) *event {
	evt, _ := req.Attribute(eventLogAttribute).(*event)
	return evt
}
