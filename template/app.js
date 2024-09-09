// DOM Elements
const localVideo = document.getElementById('localVideo');
const remoteVideo = document.getElementById('remoteVideo');
const startCallButton = document.getElementById('startCall');
const hangupCallButton = document.getElementById('hangupCall');

const WebSocketMessageType = {
    ICECandidateType: 1,
    Offer: 2,
    AnswerType: 3,
    OfferRequest: 4
};

// Variables
let hasOfferRequest = false;
let localStream;
let remoteStream;
let pc;
// Get the current URL query parameters
const urlParams = new URLSearchParams(window.location.search);

// Extract the specific query parameter (e.g., 'param')
const param = urlParams.get('id');

// Build the WebSocket URL using the query parameter value
const wsUrl = "ws://" + document.location.host + "/ws/" + param;
const servers = {
    iceServers: [
        {urls: 'stun:stun.l.google.com:19302'}
    ]
};

// WebSocket Connection
let ws;

const initializeWebSocket = () => {
    ws = new WebSocket(wsUrl);

    ws.onopen = () => console.log("WebSocket connected");

    ws.onmessage = async (e) => {
        const message = JSON.parse(e.data);
        switch (message.type) {
            case WebSocketMessageType.ICECandidateType:
                await handleIceCandidate(message.data);
                break;
            case WebSocketMessageType.Offer:
                await handleRemoteOffer(message.data);
                break;
        }
    };
};

const startCall = async () => {
        initializeWebSocket();

        localStream = await navigator.mediaDevices.getUserMedia({video: true, audio: true});
        localVideo.srcObject = localStream;

        pc = new RTCPeerConnection(servers);
        pc.onconnectionstatechange = e => {
            console.log("connection state: ", e.currentTarget.connectionState);
        }
        pc.onnegotiationneeded = handleNegotiation;
        pc.onsignalingstatechange = async (e) => {
            if (e.currentTarget.signalingState === "stable") {
                if (hasOfferRequest) {
                    hasOfferRequest = false;
                    let offer = await pc.createOffer();
                    ws.send(JSON.stringify({type: WebSocketMessageType.Offer, data: offer}));
                }
            }
        }
        addLocalTracks();
        setupPeerConnectionListeners();
    }
;

const addLocalTracks = () => {
    localStream.getTracks().forEach(track => {
        pc.addTrack(track, localStream);
    });
};

const setupPeerConnectionListeners = () => {
    pc.onicecandidate = (event) => {
        if (event.candidate) {
            ws.send(JSON.stringify({type: WebSocketMessageType.ICECandidateType, data: event.candidate.toJSON()}));
        }
    };

    pc.ontrack = (event) => {
        console.log("OnTrack")
        remoteVideo.srcObject = event.streams[0];
    };
};

const handleNegotiation = () => {
    if (!hasOfferRequest) {
        hasOfferRequest = true;
    }
};

const handleIceCandidate = async (candidate) => {
    try {
        await pc.addIceCandidate(candidate);
        console.log("ICE candidate added");
    } catch (error) {
        console.error("Error adding ICE candidate:", error);
    }
};

const handleRemoteOffer = async (offer) => {
    try {
        setTimeout(async () => {
            await pc.setRemoteDescription(offer);
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            ws.send(JSON.stringify({type: WebSocketMessageType.AnswerType, data: answer}));
        }, 3000)
    } catch (error) {
        console.error("Error handling remote offer:", error);
    }
};

const hangupCall = () => {
    if (pc) {
        pc.close();
        pc = null;
    }
    localVideo.srcObject = null;
    remoteVideo.srcObject = null;
};

// Event Listeners
startCallButton.onclick = startCall;
hangupCallButton.onclick = hangupCall;