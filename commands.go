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
package aves

import (
	cmds "github.com/maarek/aves/commands"
	"github.com/maarek/aves/commands/events"
	"github.com/maarek/aves/commands/pubsub"
	"github.com/maarek/aves/commands/stream"
)

type Command string

const (
	STREAM_DELETE Command = "delete"
	STREAM_EXISTS Command = "exists"
	STREAM_LIST   Command = "slist"

	EVENT_LIST Command = "elist"

	EVENT_PUBLISH    Command = "publish"
	STREAM_SUBSCRIBE Command = "subscribe"
	SUBSCRIBE_ALL    Command = "subscribeall"
)

var (
	// server receive handlers
	Commands = map[Command]cmds.ReceiveHandler{
		// stream
		STREAM_DELETE: stream.DeleteCommand,
		STREAM_EXISTS: stream.ExistsCommand,
		STREAM_LIST:   stream.ListCommand,

		// events
		EVENT_LIST: events.RangeCommand,

		// pubsub
		EVENT_PUBLISH:    pubsub.PublishCommand,
		STREAM_SUBSCRIBE: pubsub.SubscribeCommand,
		SUBSCRIBE_ALL:    pubsub.SubscribeAllCommand,
	}
)
