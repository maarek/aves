/*
 * Copyright 2020 Jeremy Lyman
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package client

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
)

// Stream - defines the aggregate stream response
type Stream struct {
	StreamID   string
	EventCount int
}

func parseSListResp(resp []interface{}) ([]Stream, error) {
	var streams []Stream
	if err := redis.ScanSlice(resp, &streams); err != nil {
		return nil, fmt.Errorf("error parsing streams")
	}

	return streams, nil
}

// SimpleEvent - defines an event on a stream without metadata
type SimpleEvent struct {
	Version int
	Data    string
}

func parseSimpleEventListResp(resp []interface{}) ([]SimpleEvent, error) {
	var events []SimpleEvent
	if err := redis.ScanSlice(resp, &events); err != nil {
		return nil, fmt.Errorf("error parsing events")
	}

	return events, nil
}

// FullEvent - defines an event on a stream with metadata
type FullEvent struct {
	StreamID string
	EventID  string
	Version  int
	Data     string
}

func parseFullEventListResp(resp []interface{}) ([]FullEvent, error) {
	var events []FullEvent
	if err := redis.ScanSlice(resp, &events); err != nil {
		return nil, fmt.Errorf("error parsing events")
	}

	return events, nil
}
