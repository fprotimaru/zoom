package call

import (
	"encoding/json"
	"io"
	"slices"
	"zoom/internal/entity"
	"zoom/internal/pkg/wsconn"

	"github.com/alphadose/haxmap"
	"github.com/charmbracelet/log"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/webrtc/v3"
)

type Call struct {
	rtcClientAPI        *haxmap.Map[string, *webrtc.API]
	rtcClients          *haxmap.Map[string, *webrtc.PeerConnection]
	rtcClientTracks     *haxmap.Map[string, *webrtc.TrackLocalStaticRTP]
	rtcClientRTPSenders *haxmap.Map[string, *webrtc.RTPSender]
	wsClients           *haxmap.Map[string, *wsconn.WSConn]
}

func NewCall() *Call {
	return &Call{
		rtcClientAPI:        haxmap.New[string, *webrtc.API](),
		rtcClients:          haxmap.New[string, *webrtc.PeerConnection](),
		rtcClientTracks:     haxmap.New[string, *webrtc.TrackLocalStaticRTP](),
		rtcClientRTPSenders: haxmap.New[string, *webrtc.RTPSender](),
		wsClients:           haxmap.New[string, *wsconn.WSConn](),
	}
}

func (c *Call) AddClient(id string, codec webrtc.RTPCodecType) error {
	me := &webrtc.MediaEngine{}
	if err := me.RegisterDefaultCodecs(); err != nil {
		return err
	}

	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(me, i); err != nil {
		return err
	}

	intervalPliFactory, err := intervalpli.NewReceiverInterceptor()
	if err != nil {
		return err
	}
	i.Add(intervalPliFactory)

	s := webrtc.SettingEngine{}
	s.SetReceiveMTU(65000)

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(me),
		webrtc.WithInterceptorRegistry(i),
		webrtc.WithSettingEngine(s),
	)

	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return err
	}

	_, err = pc.AddTransceiverFromKind(codec, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionSendrecv,
	})
	if err != nil {
		return err
	}

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) { c.onConnectionStateChange(id, state) })
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) { c.onICEConnectionStateChange(id, state) })
	pc.OnSignalingStateChange(func(state webrtc.SignalingState) { c.onSignalingStateChange(id, state) })
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) { c.onICECandidate(id, candidate) })
	pc.OnTrack(func(t *webrtc.TrackRemote, r *webrtc.RTPReceiver) { c.onTrack(id, t, r) })

	c.rtcClients.Set(id, pc)

	return nil
}

func (c *Call) onConnectionStateChange(id string, state webrtc.PeerConnectionState) {
	log.Infof("onConnectionStateChange: %s", state)
	switch state {
	case webrtc.PeerConnectionStateConnected:
		c.rtcClients.ForEach(func(_id string, pc *webrtc.PeerConnection) bool {
			if _id == id {
				return true
			}

			track, ok := c.rtcClientTracks.Get(id)
			if !ok {
				return false
			}

			_, err := pc.AddTrack(track)
			if err != nil {
				log.Errorf("add track: %v", err)
				return false
			}

			return true
		})
	case webrtc.PeerConnectionStateFailed:
		fallthrough
	case webrtc.PeerConnectionStateClosed:
		// delete user conn
		// if err := c.Close(); err != nil {
		// 	log.Error(err)
		// 	return
		// }
	default:
	}
}

func (c *Call) onICEConnectionStateChange(id string, state webrtc.ICEConnectionState) {
	log.Infof("onICEConnectionStateChange: %s", state)
}

func (c *Call) onSignalingStateChange(id string, state webrtc.SignalingState) {
	log.Infof("onSignalingStateChange: %s", state)
}

func (c *Call) onICECandidate(id string, candidate *webrtc.ICECandidate) {
	log.Info("onICECandidate")
	if candidate == nil {
		log.Infof("candidate: %s is nil. skipping...", id)
		return
	}

	candidateInit := candidate.ToJSON()
	wsClient, ok := c.wsClients.Get(id)
	if !ok {
		log.Errorf("wsclient: %s not found", id)
		return
	}

	data, _ := json.Marshal(candidateInit)
	wsClient.SendMessage(&entity.WebSocketMessage{
		Type: 0,
		Data: data,
	})
}

func (c *Call) onTrack(id string, t *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
	log.Info("onTrack")
	track, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, id, id)
	if err != nil {
		log.Errorf("%s create a new RTP track: %v", id, err)
		return
	}

	c.addTrackClients(track, id)

	c.rtcClientTracks.Set(id, track)

	buf := make([]byte, 1400)
	for {
		n, _, err := t.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Error(err)
			}
			break
		}

		_, err = track.Write(buf[:n])
		if err != nil {
			break
		}
	}
}

func (c *Call) addTrackClients(t *webrtc.TrackLocalStaticRTP, excludeClients ...string) {
	c.rtcClients.ForEach(func(id string, pc *webrtc.PeerConnection) bool {
		if slices.Contains(excludeClients, id) {
			return true
		}
		_, err := pc.AddTrack(t)
		if err != nil {
			log.Errorf("Add track to %s: %v", id, err)
			return true
		}
		offer, err := pc.CreateOffer(nil)
		if err != nil {
			log.Error(err)
			return true
		}
		if err := pc.SetLocalDescription(offer); err != nil {
			log.Error(err)
			return true
		}
		data, _ := json.Marshal(offer)
		ws, _ := c.wsClients.Get(id)
		ws.SendMessage(&entity.WebSocketMessage{
			Type: entity.Offer,
			Data: data,
		})

		return true
	})
}
