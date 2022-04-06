// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	webrtc "github.com/pion/webrtc/v3"

	"log"
	"os"
	"os/signal"
	"time"
)

//wss://ienso.ienso-dev.com/api/signaling?accessToken=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhxMDM1YmV4Y28zT0tfQVhHQm4wOXpjcXJuMXVNcnNQZGxqMlNHdlI4cWMifQ.eyJzaWQiOiI3YzAzNzJmYy0wZmE4LTRjNDItYTBkZi1lYzAwYTQ5MTMzYjciLCJpYXQiOjE2NDkyNTY5OTUsImV4cCI6MTY0OTg2MTc5NSwiYXVkIjpbXSwiaXNzIjoiZGV2aWNlLXNpZ25hbGluZy1hdXRob3JpdHkifQ.L-Xej8ezKR9Uo75SS3OTgJJJQ8LrMpqZPACAvTCGmwRrLbF7URO-LDwDCd4Hqvb174ZWKIQ3WRz1PYecvN16uPywRZuNCpRypalI1aeZTfSZ5K5TPgbCjBwCJ9DoDngCau-eE5Tjvl06vYicVtv6FeY_FNItInkmhSUIdPcKEl_TJ-5bqbzY9-yW1DL2UGGjomnqaGlOsXv0eCJp9mlAOW0F6MpA2ym9uDpqik1MqThXLf3_d11g2Ybec-nctCvbN8L3GLf2V4okaQThYb8Wz1lcc3jh8Bbw_krCF1KXMnnkIuq0A1H7DpRFeJXdJfz27EMkOEAz8dBoMBLcdzQR8Q

//wss://demo.piesocket.com/v3/channel_1?api_key=oCdCMcMPQpbvNjUIzqtvF1d2X2okWpDQj4AwARJuAgtjhzKxVEjQU6IdCjwm&notify_self

// SessionDescription is used to expose local and remote session descriptions.
type ResponseAnswerStruct struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func websocketClient() {

	var clientUrl = Config.GetWSClientUrl()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ws, _, err := websocket.DefaultDialer.Dial(clientUrl, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer ws.Close()
	muxerWebRTC := NewLMuxer(Options{ICEServers: Config.GetICEServers(), ICEUsername: Config.GetICEUsername(), ICECredential: Config.GetICECredential(), PortMin: Config.GetWebRTCPortMin(), PortMax: Config.GetWebRTCPortMax()})
	done := make(chan struct{})

	go func() {
		defer close(done)
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
			case json.Unmarshal(message, &offer) == nil && offer.SDP != "" && offer.Type == webrtc.SDPTypeOffer:
				log.Println("Received offer")
				codecs := Config.coGe("H264_AAC")
				answer, err, _ := muxerWebRTC.WriteHeader(codecs, offer.SDP, func(c *webrtc.ICECandidate) {
					log.Println("Generated Candidate")
					if c == nil {
						return
					}
					o, marshalErr := json.Marshal(c.ToJSON())
					if marshalErr != nil {
						panic(marshalErr)
					}
					if err = ws.WriteMessage(websocket.TextMessage, o); err != nil {
						panic(err)
					}
				})

				if err != nil {
					log.Println("Muxer WriteHeader", err)
					return
				}
				outbound, marshalErr := json.Marshal(ResponseAnswerStruct{
					Type:    "answer",
					Payload: string(answer),
				})
				if marshalErr != nil {
					panic(marshalErr)
				}
				log.Println("Sent Answer")
				if err = ws.WriteMessage(websocket.TextMessage, outbound); err != nil {
					panic(err)
				}
				go func() {
					_, ch := Config.clAd("H264_AAC")
					defer muxerWebRTC.Close()
					var videoStart bool
					noVideo := time.NewTimer(10 * time.Second)
					for {
						select {
						case <-noVideo.C:
							log.Println("noVideo")
							return
						case pck := <-ch:
							if pck.IsKeyFrame {
								noVideo.Reset(10 * time.Second)
								videoStart = true
							}
							if !videoStart {
								continue
							}
							err = muxerWebRTC.WritePacket(pck)
							if err != nil {
								log.Println("WritePacket", err)
								return
							}
						}
					}
				}()

			case json.Unmarshal(message, &candidate) == nil && candidate.Candidate != "":
				log.Println("Received Candidate ", candidate)
				if err = muxerWebRTC.pc.AddICECandidate(candidate); err != nil {
					panic(err)
				}

			default:
				log.Printf("GOT:: %s", message)
				// panic("Unknown message")
			}

		}
	}()

	//buf := make([]byte, 1500)

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
