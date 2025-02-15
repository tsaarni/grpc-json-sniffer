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

    renderMessageList() {
        const list = this.messagesListContainer;
        const filterValue = this.filterInput.value.toLowerCase();

        list.innerHTML = "";
        this.messages.forEach((msg, index) => {
            if (!msg.method.toLowerCase().includes(filterValue) && !msg.type.toLowerCase().includes(filterValue)) {
                return;
            }

            const item = this.messageListTemplate.cloneNode(true);
            const numberSpan = item.querySelector('.message-row-number');
            const timestampSpan = item.querySelector('.message-row-timestamp');
            const methodTypeSpan = item.querySelector('.message-row-methodtype');

            numberSpan.textContent = msg.id;
            timestampSpan.textContent = this.formatTimestamp(msg.time);
            methodTypeSpan.textContent = `${msg.method} (${msg.type})`;
            if (msg.direction === "recv") {
                item.classList.add("recv");
            } else if (msg.direction === "send") {
                item.classList.add("send");
            }

            item.addEventListener("click", () => {
                this.selectedMessageId = msg.id; // Update selected message ID
                this.messagesListContainer.querySelectorAll(".message-row-content").forEach((el) => el.classList.remove("selected"));
                item.classList.add("selected");
                this.renderMessageDetails(msg);
            });

            list.appendChild(item);

            if (msg.id === this.selectedMessageId) {
                item.classList.add("selected");
            }
        });
    }

    renderMessageDetails(msg) {
        const details = this.messageDetailsTemplate.cloneNode(true);

        details.querySelector('#message-details-id-value').textContent = msg.id;
        details.querySelector('#message-details-timestamp-value').textContent = this.formatTimestamp(msg.time);
        details.querySelector('#message-details-method-value').textContent = msg.method;
        details.querySelector('#message-details-type-value').textContent = msg.type;
        details.querySelector('#message-details-direction-value').textContent = msg.direction;
        details.querySelector('#message-details-payload-value').textContent = JSON.stringify(msg.message, null, 2);

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
            if (msg.id == 1) {
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
