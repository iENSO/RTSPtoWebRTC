// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	deepchWebrtc "github.com/deepch/vdk/format/webrtcv3"
	"github.com/gorilla/websocket"
	webrtc "github.com/pion/webrtc/v3"

	"log"
	"os"
	"os/signal"
	"time"
)

//wss://ienso.ienso-dev.com/api/signaling?accessToken=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhxMDM1YmV4Y28zT0tfQVhHQm4wOXpjcXJuMXVNcnNQZGxqMlNHdlI4cWMifQ.eyJzaWQiOiI3YzAzNzJmYy0wZmE4LTRjNDItYTBkZi1lYzAwYTQ5MTMzYjciLCJpYXQiOjE2NDkyNTY5OTUsImV4cCI6MTY0OTg2MTc5NSwiYXVkIjpbXSwiaXNzIjoiZGV2aWNlLXNpZ25hbGluZy1hdXRob3JpdHkifQ.L-Xej8ezKR9Uo75SS3OTgJJJQ8LrMpqZPACAvTCGmwRrLbF7URO-LDwDCd4Hqvb174ZWKIQ3WRz1PYecvN16uPywRZuNCpRypalI1aeZTfSZ5K5TPgbCjBwCJ9DoDngCau-eE5Tjvl06vYicVtv6FeY_FNItInkmhSUIdPcKEl_TJ-5bqbzY9-yW1DL2UGGjomnqaGlOsXv0eCJp9mlAOW0F6MpA2ym9uDpqik1MqThXLf3_d11g2Ybec-nctCvbN8L3GLf2V4okaQThYb8Wz1lcc3jh8Bbw_krCF1KXMnnkIuq0A1H7DpRFeJXdJfz27EMkOEAz8dBoMBLcdzQR8Q

//wss://demo.piesocket.com/v3/channel_1?api_key=oCdCMcMPQpbvNjUIzqtvF1d2X2okWpDQj4AwARJuAgtjhzKxVEjQU6IdCjwm&notify_self

func websocketClient() {

	var clientUrl = Config.GetWSClientUrl()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ws, _, err := websocket.DefaultDialer.Dial(clientUrl, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer ws.Close()

	var codecs []JCodec
	codecs = append(codecs, JCodec{Type: "video"})

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	muxerWebRTC := deepchWebrtc.NewMuxer(deepchWebrtc.Options{ICEServers: Config.GetICEServers(), ICEUsername: Config.GetICEUsername(), ICECredential: Config.GetICECredential(), PortMin: Config.GetWebRTCPortMin(), PortMax: Config.GetWebRTCPortMax()})

	peerConnection, err := muxerWebRTC.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		log.Println("Got Ice Candidate")
		if c == nil {
			return
		}

		outbound, marshalErr := json.Marshal(c.ToJSON())
		if marshalErr != nil {
			panic(marshalErr)
		}

		if err = ws.WriteMessage(websocket.TextMessage, outbound); err != nil {
			panic(err)
		}
	})

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Send the current time via a DataChannel to the remote peer every 3 seconds
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnOpen(func() {
			for range time.Tick(time.Second * 3) {
				if err = d.SendText(time.Now().String()); err != nil {
					panic(err)
				}
			}
		})
	})

	done := make(chan struct{})

	//buf := make([]byte, 1500)
	for {
		// Read each inbound WebSocket Message
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		// Unmarshal each inbound WebSocket message
		var (
			candidate webrtc.ICECandidateInit
			offer     webrtc.SessionDescription
		)

		switch {

		// Attempt to unmarshal as a SessionDescription. If the SDP field is empty
		// assume it is not one.
		case json.Unmarshal(message, &offer) == nil && offer.SDP != "":
			if err = peerConnection.SetRemoteDescription(offer); err != nil {
				panic(err)
			}

			answer, answerErr := peerConnection.CreateAnswer(nil)
			if answerErr != nil {
				panic(answerErr)
			}

			if err = peerConnection.SetLocalDescription(answer); err != nil {
				panic(err)
			}

			outbound, marshalErr := json.Marshal(answer)
			if marshalErr != nil {
				panic(marshalErr)
			}

			if err = ws.WriteMessage(websocket.TextMessage, outbound); err != nil {
				panic(err)
			}
		// Attempt to unmarshal as a ICECandidateInit. If the candidate field is empty
		// assume it is not one.
		case json.Unmarshal(message, &candidate) == nil && candidate.Candidate != "":
			if err = peerConnection.AddICECandidate(candidate); err != nil {
				panic(err)
			}
		default:
			log.Printf("GOT:: %s", message)
			// panic("Unknown message")
		}
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := ws.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
