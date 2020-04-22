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

package oplog

type broadcast struct {
	c chan broadcast
	v interface{}
}

// Broadcaster - defines a pubsub mechanism for distributing events to listeners
type Broadcaster struct {
	listenc chan chan (chan broadcast)
	sendc   chan<- interface{}
}

// Receiver - a listener for events
type Receiver struct {
	c chan broadcast
}

// NewBroadcaster - create a new broadcaster object.
func NewBroadcaster() Broadcaster {
	listenc := make(chan (chan (chan broadcast)))
	sendc := make(chan interface{})
	go func() {
		currc := make(chan broadcast, 1)
		for {
			select {
			case v := <-sendc:
				if v == nil {
					currc <- broadcast{}
					return
				}
				c := make(chan broadcast, 1)
				b := broadcast{c: c, v: v}
				currc <- b
				currc = c
			case r := <-listenc:
				r <- currc
			}
		}
	}()
	return Broadcaster{
		listenc: listenc,
		sendc:   sendc,
	}
}

// Listen - start listening to the broadcasts.
func (b Broadcaster) Listen() Receiver {
	c := make(chan chan broadcast)
	b.listenc <- c
	return Receiver{<-c}
}

// Write - broadcast a value to all listeners.
func (b Broadcaster) Write(v interface{}) { b.sendc <- v }

// Read - read a value that has been broadcast,
// waiting until one is available if necessary.
func (r *Receiver) Read() interface{} {
	b := <-r.c
	v := b.v
	r.c <- b
	r.c = b.c
	return v
}
