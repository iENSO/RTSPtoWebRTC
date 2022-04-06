let socket;
let connections = {}
let stream = new MediaStream();

const generateSessionToken = async () => {
    return `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhxMDM1YmV4Y28zT0tfQVhHQm4wOXpjcXJuMXVNcnNQZGxqMlNHdlI4cWMifQ.eyJzaWQiOiJjZDVkZTVlNC02ZmU1LTQxYmYtYjVlMy1iNTVlNzQ3YWRlZjAiLCJpYXQiOjE2NDkyNjUyNzcsImV4cCI6MTY0OTg3MDA3NywiYXVkIjpbXSwiaXNzIjoiZGV2aWNlLXNpZ25hbGluZy1hdXRob3JpdHkifQ.bmx5qY5NzritAYHPVuMSBTEqSZ2SdNb2lsuRllhEAnupDGZ8wtOK6_YR51mBSgeJeEkpMCEKdLO6BBYGTpu03o1_FgZjkircEPB7TqsSNJFAjguTWWmk9bom_tAmsJ2L8Giy_K-r1uANMjAj09zHTiKe-0U9_fta2f7_BpTlUC3UP2cHAc5rAykd7qP-9uDuGEoYDDHcCBIjGbk10dXWpQ5NWle9ibcnXftonVSf1iUc-k67nSiZGsUPWluskQEJgKPly2yJqwUqJGuxzh9eQO2dJkvUUZV6Toub8GfrC2MxadqxBg-WAI0l0BQS-pe6h7rBLAKL8-_XwGo2yRFQXw`;
}

const startListening = (iensoWssToken) => {
    iensoWssToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InhxMDM1YmV4Y28zT0tfQVhHQm4wOXpjcXJuMXVNcnNQZGxqMlNHdlI4cWMifQ.eyJzaWQiOiJjZDVkZTVlNC02ZmU1LTQxYmYtYjVlMy1iNTVlNzQ3YWRlZjAiLCJpYXQiOjE2NDkyNjUyNzcsImV4cCI6MTY0OTg3MDA3NywiYXVkIjpbXSwiaXNzIjoiZGV2aWNlLXNpZ25hbGluZy1hdXRob3JpdHkifQ.bmx5qY5NzritAYHPVuMSBTEqSZ2SdNb2lsuRllhEAnupDGZ8wtOK6_YR51mBSgeJeEkpMCEKdLO6BBYGTpu03o1_FgZjkircEPB7TqsSNJFAjguTWWmk9bom_tAmsJ2L8Giy_K-r1uANMjAj09zHTiKe-0U9_fta2f7_BpTlUC3UP2cHAc5rAykd7qP-9uDuGEoYDDHcCBIjGbk10dXWpQ5NWle9ibcnXftonVSf1iUc-k67nSiZGsUPWluskQEJgKPly2yJqwUqJGuxzh9eQO2dJkvUUZV6Toub8GfrC2MxadqxBg-WAI0l0BQS-pe6h7rBLAKL8-_XwGo2yRFQXw"
    console.log(iensoWssToken)
    socket = new WebSocket(`wss://ienso.ienso-dev.com/api/signaling?accessToken=${iensoWssToken}`);
    socket.onclose = () => console.log('disconnect from ienso websocket')
    socket.onerror = (error) => console.log(`ienso websocket error ${JSON.stringify(error)}`)
    socket.onmessage = (message) => {
        try {
            handleSignalingMessage(message, socket)
        } catch (e) {
            console.error(e)
        }
    }
    socket.onopen = async () => {
        console.log('connected to ienso websocket...');
         connections["sessionId"] = new RTCPeerConnection({
             iceServers: [{
                 urls: ["stun:stun.l.google.com:19302"]
             }]
        })
        const pc = connections["sessionId"];
        pc.addTransceiver("video", {
            'direction': 'sendrecv'
        })
        pc.ontrack = function (event) {
            console.log("AAAAAAAAAAAA(ontrack)")
            stream.addTrack(event.track);
            document.getElementById("videoElem").srcObject = stream;
        }
        pc.onnegotiationneeded = async () => {
            console.log("onnegotiationneeded")
            const offer = await pc.createOffer()
            await pc.setLocalDescription(offer)
            const encodedOffer = {
                type: offer.type,
                sdp: window.btoa(offer.sdp)
            }
            socket.send(JSON.stringify(encodedOffer));
        }
        pc.onicecandidate = ({candidate}) => {
            if (candidate) {
                  console.log("Generated ICE Candidate", candidate);
                  socket.send(JSON.stringify(candidate));
            }
        }
    }

}



const handleSignalingOffer = async (sessionId, offer) => {
    const pc = connections[sessionId]
    console.log("handleSignalingAnswer", offer);
    const decodedOffer = {
        type: offer.type,
        sdp: window.atob(offer.sdp)
    }
    await pc.setRemoteDescription(decodedOffer);

return;
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);
    const encodedAnswer = {
        type: answer.type,
        sdp: window.btoa(answer.sdp)
    }
    console.log("Sending Answer", answer)
    socket.send(JSON.stringify(encodedAnswer));
}

const handleSignalingAnswer = async (sessionId, answer) => {

    const pc = connections[sessionId]
    const decodedAnswer = new RTCSessionDescription({
        type: answer.type,
        sdp: window.atob(answer.sdp)
    })
    await pc.setRemoteDescription(decodedAnswer);
}

const handleSignalingCandidate = async (sessionId, candidate) => {
    return;
    const pc = connections[sessionId]
    pc.addIceCandidate(candidate);
}

const handleSignalingMessage = async (message, socket) => {
    const data  = JSON.parse(message.data);
    const {type, payload} = data;


    const sessionId = "sessionId";
    if (type === "answer" ) {
        handleSignalingAnswer(sessionId, {
            type,
            sdp: payload,
        })
        return;
    }


    if (message.data.candidate !== "") {
        handleSignalingCandidate(sessionId, data)
        return;
    }

    console.log("Unknown Message", message);
    // if (payload.type === "offer" ) {
    //     handleSignalingOffer(sessionId, payload)
    //     return;
    // }


}

generateSessionToken()
    .then(sessionToken => startListening(sessionToken));