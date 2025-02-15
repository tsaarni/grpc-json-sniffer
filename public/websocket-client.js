class WebSocketClient {
    constructor(url, onMessage) {
        this.url = url;
        this.onMessage = onMessage;
        this.socket = new WebSocket(url);

        this.socket.onopen = () => {
            console.log("WebSocket connected to " + url);
        };

        this.socket.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                this.onMessage(msg);
            } catch (e) {
                console.error("Failed to parse message", e);
            }
        };

        this.socket.onclose = (event) => {
            if (event.wasClean) {
                console.log(`[close] Connection closed cleanly, code=${event.code} reason=${event.reason}`);
            } else {
                console.log('[close] Connection died');
            }
        };

        this.socket.onerror = (error) => {
            console.log("WebSocket error: " + error);
        };
    }
}
