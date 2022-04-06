let connections = {}

const generateSessionToken = async () => {
    return `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhxMDM1YmV4Y28zT0tfQVhHQm4wOXpjcXJuMXVNcnNQZGxqMlNHdlI4cWMifQ.eyJzaWQiOiJjZDVkZTVlNC02ZmU1LTQxYmYtYjVlMy1iNTVlNzQ3YWRlZjAiLCJpYXQiOjE2NDkyNjUyNzcsImV4cCI6MTY0OTg3MDA3NywiYXVkIjpbXSwiaXNzIjoiZGV2aWNlLXNpZ25hbGluZy1hdXRob3JpdHkifQ.bmx5qY5NzritAYHPVuMSBTEqSZ2SdNb2lsuRllhEAnupDGZ8wtOK6_YR51mBSgeJeEkpMCEKdLO6BBYGTpu03o1_FgZjkircEPB7TqsSNJFAjguTWWmk9bom_tAmsJ2L8Giy_K-r1uANMjAj09zHTiKe-0U9_fta2f7_BpTlUC3UP2cHAc5rAykd7qP-9uDuGEoYDDHcCBIjGbk10dXWpQ5NWle9ibcnXftonVSf1iUc-k67nSiZGsUPWluskQEJgKPly2yJqwUqJGuxzh9eQO2dJkvUUZV6Toub8GfrC2MxadqxBg-WAI0l0BQS-pe6h7rBLAKL8-_XwGo2yRFQXw`;
}

const startListening = (iensoWssToken) => {
    iensoWssToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhxMDM1YmV4Y28zT0tfQVhHQm4wOXpjcXJuMXVNcnNQZGxqMlNHdlI4cWMifQ.eyJzaWQiOiJjZDVkZTVlNC02ZmU1LTQxYmYtYjVlMy1iNTVlNzQ3YWRlZjAiLCJpYXQiOjE2NDkyNjUyNzcsImV4cCI6MTY0OTg3MDA3NywiYXVkIjpbXSwiaXNzIjoiZGV2aWNlLXNpZ25hbGluZy1hdXRob3JpdHkifQ.bmx5qY5NzritAYHPVuMSBTEqSZ2SdNb2lsuRllhEAnupDGZ8wtOK6_YR51mBSgeJeEkpMCEKdLO6BBYGTpu03o1_FgZjkircEPB7TqsSNJFAjguTWWmk9bom_tAmsJ2L8Giy_K-r1uANMjAj09zHTiKe-0U9_fta2f7_BpTlUC3UP2cHAc5rAykd7qP-9uDuGEoYDDHcCBIjGbk10dXWpQ5NWle9ibcnXftonVSf1iUc-k67nSiZGsUPWluskQEJgKPly2yJqwUqJGuxzh9eQO2dJkvUUZV6Toub8GfrC2MxadqxBg-WAI0l0BQS-pe6h7rBLAKL8-_XwGo2yRFQXw"
    console.log(iensoWssToken)
    const socket = new WebSocket(`wss://ienso.ienso-dev.com/api/signaling?accessToken=${iensoWssToken}`);
    socket.onclose = () => console.log('disconnect from ienso websocket')
    socket.onerror = (error) => console.log(`ienso websocket error ${JSON.stringify(error)}`)
    socket.onmessage = (message) => handleSignalingMessage(message, socket)
    socket.onopen = () => {
        console.log('connected to ienso websocket...');
        handleSignalingCall("sessionId", socket)
    }

}

const handleSignalingCall = async (sessionId, socket) => {
    connections[sessionId] = new RTCPeerConnection({
        iceServers: [{
            urls: 'stun:stun.ienso-dev.com:3478'
        }],
        "iceTransportPolicy": "all",
        "iceCandidatePoolSize": "0"
    })
    const pc = connections[sessionId]

    pc.onnegotiationneeded = async () => {
        const offer = await pc.createOffer()
        await pc.setLocalDescription(offer)
        socket.send(JSON.stringify(offer));
    }

    pc.onicecandidate = ({candidate}) => {
        if (candidate) {
            socket.send(candidate);
        }
    }

    const stream = await navigator.mediaDevices.getUserMedia({video: true})
    stream.getTracks().forEach(track => {
        pc.addTrack(track, stream)
    })
}

const handleSignalingAnswer = async (sessionId, answer) => {
    const pc = connections[sessionId]
    console.log("answeransweransweransweranswer", answer)
    await pc.setRemoteDescription(answer)
}

const handleSignalingCandidate = async (sessionId, candidate) => {
    const pc = connections[sessionId]
    pc.addIceCandidate(candidate);
}

const handleSignalingMessage = async (message, socket) => {
    const data = JSON.parse(message.data);
    console.log(data)
    const sessionId = "sessionId";
    if (data.type === "answer") {
        handleSignalingAnswer(sessionId, data)
    }
    if (data.candidate !== "") {
        handleSignalingCandidate(sessionId, data)
    }
}

generateSessionToken()
    .then(sessionToken => startListening(sessionToken));