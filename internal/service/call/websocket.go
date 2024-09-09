package call

import (
	"encoding/json"
	"time"
	"zoom/internal/entity"
	"zoom/internal/pkg/wsconn"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

func (c *Call) AddWebSocketClient(id string, conn *websocket.Conn) {
	ws := wsconn.NewWSConn(id, conn)
	ws.SetMessageCB(c.wsMessageHandler)
	ws.SetDisconnectCB(func(id string) {
		log.Infof("ws conn closed: %s", id)
	})
	c.wsClients.Set(id, ws)
	if err := c.AddClient(id, webrtc.RTPCodecTypeAudio); err != nil {
		log.Error(err)
	}
	client, ok := c.rtcClients.Get(id)
	_ = ok
	offer, err := client.CreateOffer(nil)
	if err != nil {
		log.Error(err)
		return
	}
	if err := client.SetLocalDescription(offer); err != nil {
		log.Error(1, err)
		return
	}
	data, _ := json.Marshal(offer)
	ws.SendMessage(&entity.WebSocketMessage{
		Type: entity.Offer,
		Data: data,
	})
}

func (c *Call) wsMessageHandler(id string, msg *entity.WebSocketMessage) {
	ws, ok := c.wsClients.Get(id)
	_ = ok
	client, ok := c.rtcClients.Get(id)
	if !ok {
		log.Error("client not found")
		return
	}
	switch msg.Type {
	case entity.ICECandidateType:

		var candidateInit webrtc.ICECandidateInit
		if err := json.Unmarshal(msg.Data, &candidateInit); err != nil {
			log.Error(err)
			return
		}
		if err := client.AddICECandidate(candidateInit); err != nil {
			log.Error(err)
		}
	case entity.Offer:
		var offer webrtc.SessionDescription
		if err := json.Unmarshal(msg.Data, &offer); err != nil {
			log.Error(err)
			return
		}
		if err := client.SetRemoteDescription(offer); err != nil {
			log.Error(err)
			return
		}

		answer, err := client.CreateAnswer(nil)
		if err != nil {
			log.Error(err)
			return
		}
		if err := client.SetLocalDescription(answer); err != nil {
			log.Error(err)
			return
		}

		data, _ := json.Marshal(answer)
		ws.SendMessage(&entity.WebSocketMessage{
			Type: entity.AnswerType,
			Data: data,
		})
	case entity.OfferRequest:
		time.Sleep(time.Second)
		client, _ := c.rtcClients.Get(id)
		if client.SignalingState() != webrtc.SignalingStateStable {
			log.Error("wrong state")
			return
		}

		offer, err := client.CreateOffer(nil)
		if err != nil {
			log.Error(err)
			return
		}

		if err := client.SetLocalDescription(offer); err != nil {
			log.Error(2, err)
			return
		}
		data, _ := json.Marshal(offer)
		ws.SendMessage(&entity.WebSocketMessage{
			Type: entity.Offer,
			Data: data,
		})
	case entity.AnswerType:
		client, _ := c.rtcClients.Get(id)
		var answer webrtc.SessionDescription
		if err := json.Unmarshal(msg.Data, &answer); err != nil {
			log.Error(err)
			return
		}
		if err := client.SetRemoteDescription(answer); err != nil {
			log.Error(err)
			return
		}
		log.Infof("%s answer is set!", id)
	}
}
