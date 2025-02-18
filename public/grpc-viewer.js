class GrpcViewer {
    constructor() {
        this.target = document.getElementById('grpc-viewer');
        this.messages = [];

        // Templates
        this.messageListTemplate = document.getElementById('message-list-template').content.firstElementChild;
        this.messageDetailsTemplate = document.getElementById('message-details-template').content.firstElementChild;

        // Elements
        this.filterInput = document.getElementById('message-list-filter-input');
        this.clearButton = document.getElementById('message-list-clear-button');
        this.messagesListContainer = document.getElementById('messages-list-container');
        this.messageListPanel = document.getElementById('message-list-panel');
        this.resizer = document.getElementById('resizer');
        this.messageDetails = document.getElementById('message-details');
        this.detailsContent = document.getElementById('details-content');

        this.initializeEventListeners();
        this.initializeResizer();
        this.initializeWebSocket();

        this.selectedMessageId = null;
    }

    formatTimestamp(timeStamp) {
        return new Date(timeStamp).toLocaleTimeString(undefined, {
            hour: "2-digit",
            minute: "2-digit",
            second: "2-digit",
            fractionalSecondDigits: 3,
            hour12: false,
        });
    }

    stripNamespace(method) {
        const parts1 = method.split("/");
        const lastPart = parts1[parts1.length - 1];
        const parts2 = lastPart.split(".");
        return parts2[parts2.length - 1];
    }

    matchesFilter(msg, filter) {
        if (!filter) return true;
        if (filter.includes(":")) {
            const parts = filter.split(":");
            if (parts.length >= 2) {
                const key = parts[0].trim().toLowerCase();
                const value = parts[1].trim().toLowerCase();
                if (key in msg) {
                    return String(msg[key]).toLowerCase() == value;
                }
            }
        }
        return Object.values(msg).some(val => String(val).toLowerCase().includes(filter));
    }

    renderMessageList() {
        const list = this.messagesListContainer;
        const filterQuery = this.filterInput.value.trim().toLowerCase();
        list.innerHTML = "";

        this.messages.forEach((msg, index) => {
            if (!this.matchesFilter(msg, filterQuery)) {
                return;
            }

            const item = this.messageListTemplate.cloneNode(true);

            item.querySelector('.message-row-message-id').textContent = msg.message_id;
            item.querySelector('.message-row-timestamp').textContent = this.formatTimestamp(msg.time);
            item.querySelector('.message-row-method-and-message').textContent = `${this.stripNamespace(msg.method)} (${this.stripNamespace(msg.message)})`;

            if (msg.direction === "recv") {
                item.classList.add("recv");
            } else if (msg.direction === "send") {
                item.classList.add("send");
            }

            item.addEventListener("click", () => {
                this.selectedMessageId = msg.message_id;
                this.messagesListContainer.querySelectorAll(".message-row-content").forEach((el) => el.classList.remove("selected"));
                item.classList.add("selected");
                this.renderMessageDetails(msg);
            });

            list.appendChild(item);

            if (msg.message_id === this.selectedMessageId) {
                item.classList.add("selected");
            }
        });
    }

    renderMessageDetails(msg) {
        const details = this.messageDetailsTemplate.cloneNode(true);

        details.querySelector('#message-details-message-id-value').textContent = msg.message_id;
        details.querySelector('#message-details-timestamp-value').textContent = this.formatTimestamp(msg.time);
        details.querySelector('#message-details-method-value').textContent = msg.method;
        details.querySelector('#message-details-message-value').textContent = msg.message;
        details.querySelector('#message-details-direction-value').textContent = msg.direction;
        details.querySelector('#message-details-peer-address-value').textContent = msg.peer_address;
        details.querySelector('#message-details-error-value').textContent = msg.error;
        details.querySelector('#message-details-payload-value').textContent = JSON.stringify(msg.content, null, 2);

        // Optional fields.
        if ("stream_id" in msg) {
            details.querySelector('#message-details-stream-id').classList.remove("hidden");
            details.querySelector('#message-details-stream-id-value').textContent = msg.stream_id;
        }
        if ("error" in msg) {
            details.querySelector('#message-details-error').classList.remove("hidden");
            details.querySelector('#message-details-error-value').textContent = msg.error;
        }

        this.detailsContent.innerHTML = '';
        this.detailsContent.appendChild(details);
    }

    clearMessages() {
        this.messages = [];
        this.renderMessageList();
        this.detailsContent.textContent = "Select a message to view details";
        this.selectedMessageId = null;
    }

    initializeEventListeners() {
        this.filterInput.addEventListener("input", () => {
            this.renderMessageList();
        });

        this.clearButton.addEventListener("click", () => {
            this.clearMessages();
        });
    }

    initializeWebSocket() {
        this.socketClient = new WebSocketClient("ws://localhost:8080/messages", (msg) => {
            if (msg.message_id == 1) {
                this.messages = [msg]
            } else {
                this.messages.push(msg);
            }
            this.renderMessageList();
        });
    }

    initializeResizer() {
        let isResizing = false;
        const minWidth = 100;

        this.resizer.addEventListener("mousedown", (e) => {
            isResizing = true;
        });

        document.addEventListener("mousemove", (e) => {
            if (!isResizing) return;
            let newWidth = e.clientX - this.messageListPanel.offsetLeft;
            if (newWidth < minWidth) newWidth = minWidth;
            this.messageListPanel.style.width = newWidth + "px";
        });

        document.addEventListener("mouseup", (e) => {
            isResizing = false;
        });
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const app = new GrpcViewer();
});
