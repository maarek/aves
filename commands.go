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

// Command - a server command
type Command string

const (
	// StreamDelete - redis delete command
	StreamDelete Command = "delete"
	// StreamExists - redis exists command
	StreamExists Command = "exists"
	// StreamList - redis list command
	StreamList Command = "slist"

	// EventList - redis event list command
	EventList Command = "elist"

	// EventPublish - redis event publish command
	EventPublish Command = "publish"
	// StreamSubscribe - redis stream subscription command
	StreamSubscribe Command = "subscribe"
	// SubscribeAll - redis all event subscription command
	SubscribeAll Command = "subscribeall"
)

var (
	// Commands - server receive handlers
	Commands = map[Command]cmds.ReceiveHandler{
		// stream
		StreamDelete: stream.DeleteCommand,
		StreamExists: stream.ExistsCommand,
		StreamList:   stream.ListCommand,

		// events
		EventList: events.RangeCommand,

		// pubsub
		EventPublish:    pubsub.PublishCommand,
		StreamSubscribe: pubsub.SubscribeCommand,
		SubscribeAll:    pubsub.SubscribeAllCommand,
	}
)
