import { WebSocketClient } from "./websocket-client.js";
import { Environment } from "./cel.min.js";

export class GrpcViewer {
    constructor() {
        this.messages = [];

        // Templates
        this.messageListTemplate = document.getElementById("message-list-template").content.firstElementChild;
        this.messageDetailsTemplate = document.getElementById("message-details-template").content.firstElementChild;

        // Elements
        this.filterInput = document.getElementById("message-list-filter-input");
        this.filterInputClearButton = document.getElementById("filter-clear-button");
        this.filterErrorTooltip = document.getElementById("filter-error-tooltip");
        this.filterHelpLink = document.getElementById("filter-help-link");
        this.filterHelpPopup = document.getElementById("filter-help-popup");
        this.filterHelpCloseButton = document.getElementById("filter-help-close-button");
        this.clearButton = document.getElementById("message-list-clear-button");
        this.messagesListContainer = document.getElementById("messages-list-container");
        this.messageListPanel = document.getElementById("message-list-panel");
        this.resizer = document.getElementById("resizer");
        this.messageDetails = document.getElementById("message-details");
        this.detailsContent = document.getElementById("details-content");
        this.timezoneCheckbox = document.getElementById("timezone");

        // Timer to throttle message list updates while incoming messages arrive from the server or when the filter changes.
        this.renderTimer = null;

        this.timeZone = localStorage.getItem("grpc-viewer-timezone") || undefined;

        this.initializeEventListeners();
        this.initializeResizer();
        this.initializeWebSocket();

        this.selectedMessageId = null;

        // Initialize CEL environment with type-safe variable declarations
        this.celEnv = new Environment()
            .registerVariable("message_id", "dyn")
            .registerVariable("stream_id", "dyn")
            .registerVariable("direction", "string")
            .registerVariable("time", "string")
            .registerVariable("method", "string")
            .registerVariable("message", "string")
            .registerVariable("peer_address", "string")
            .registerVariable("content", "dyn")
            .registerVariable("error", "string");
    }

    getFilteredMessages() {
        const filterQuery = this.filterInput.value.trim();
        if (!filterQuery) {
            this.filterInput.classList.remove("filter-error");
            this.filterErrorTooltip.classList.add("hidden");
            this.filterErrorMessage = null;
            return this.messages;
        }

        let parsedExpr;
        try {
            parsedExpr = this.celEnv.parse(filterQuery);

            const typeCheck = this.celEnv.check(filterQuery);
            if (!typeCheck.valid) {
                this.filterInput.classList.add("filter-error");
                this.filterErrorTooltip.textContent = typeCheck.error.message;
                return [];
            }
            this.filterInput.classList.remove("filter-error");
            this.filterErrorTooltip.classList.add("hidden");
        } catch (error) {
            this.filterInput.classList.add("filter-error");
            this.filterErrorTooltip.textContent = error.message;
            return [];
        }

        return this.messages.filter(msg => {
            try {
                const result = parsedExpr(msg);
                return Boolean(result);
            } catch (error) {
                console.error("CEL evaluation error:", error.message);
                return false;
            }
        });
    }

    renderMessageList() {
        const list = this.messagesListContainer;
        const filteredMessages = this.getFilteredMessages();
        list.innerHTML = "";

        for (const msg of filteredMessages) {
            const item = this.messageListTemplate.cloneNode(true);

            item.querySelector(".message-row-message-id").textContent = msg.message_id;
            item.querySelector(".message-row-timestamp").textContent = formatTimestamp(msg.time, this.timeZone);
            item.querySelector(".message-row-method-and-message").textContent = `${stripNamespace(msg.method)} (${stripNamespace(msg.message)})`;

            if (msg.direction === "recv") {
                item.classList.add("recv");
            } else if (msg.direction === "send") {
                item.classList.add("send");
            }

            if (msg.error) {
                item.classList.add("error");
            }

            item.addEventListener("click", () => {
                this.selectedMessageId = msg.message_id;
                for (const el of this.messagesListContainer.querySelectorAll(".message-row-content")) {
                    el.classList.remove("selected");
                }
                item.classList.add("selected");
                this.renderMessageDetails(msg);
            });

            list.appendChild(item);

            if (msg.message_id === this.selectedMessageId) {
                item.classList.add("selected");
            }
        }
    }

    renderMessageDetails(msg) {
        const details = this.messageDetailsTemplate.cloneNode(true);

        details.querySelector("#message-details-message-id-value").textContent = msg.message_id;
        details.querySelector("#message-details-timestamp-value").textContent = formatTimestamp(msg.time, this.timeZone);
        details.querySelector("#message-details-method-value").appendChild(this.createFilterLink("method", msg.method));
        details.querySelector("#message-details-message-value").appendChild(this.createFilterLink("message", msg.message));
        details.querySelector("#message-details-direction-value").appendChild(this.createFilterLink("direction", msg.direction));
        details.querySelector("#message-details-peer-address-value").appendChild(this.createFilterLink("peer_address", msg.peer_address));
        details.querySelector("#message-details-payload-value").textContent = JSON.stringify(msg.content, null, 2);

        // Optional fields.
        if ("stream_id" in msg) {
            details.querySelector("#message-details-stream-id").classList.remove("hidden");
            details.querySelector("#message-details-stream-id-value").appendChild(this.createFilterLink("stream_id", msg.stream_id));
        }
        if ("error" in msg) {
            details.querySelector("#message-details-error").classList.remove("hidden");
            const error = this.createFilterLink("error", msg.error);
            error.classList.add("error");
            details.querySelector("#message-details-error-value").appendChild(error);
        }

        this.detailsContent.innerHTML = "";
        this.detailsContent.appendChild(details);
    }

    createFilterLink(key, value) {
        const link = document.createElement("a");
        link.href = "#";
        link.textContent = value;
        link.addEventListener("click", (event) => {
            event.preventDefault();
            this.applyFilter(key, value);
        });
        return link;
    }

    clearMessages() {
        this.messages = [];
        this.renderMessageList();
        this.detailsContent.textContent = "Select a message to view details";
        this.selectedMessageId = null;
    }

    initializeEventListeners() {
        this.filterInput.addEventListener("input", () => {
            this.delayedRenderMessageList();
        });

        this.filterInputClearButton.addEventListener("click", () => {
            this.filterInput.value = "";
            this.delayedRenderMessageList();
        });

        this.clearButton.addEventListener("click", () => {
            this.clearMessages();
        });

        document.addEventListener("keydown", (event) => {
            if (event.key === "ArrowUp" || event.key === "ArrowDown") {
                event.preventDefault();
                this.handleArrowKey(event);
            }
        });

        this.timezoneCheckbox.checked = this.timeZone === "UTC";
        this.timezoneCheckbox.addEventListener("change", () => {
            if (this.timezoneCheckbox.checked) {
                this.timeZone = "UTC";
                localStorage.setItem("grpc-viewer-timezone", this.timeZone);
            } else {
                this.timeZone = undefined;
                localStorage.removeItem("grpc-viewer-timezone");
            }
            this.delayedRenderMessageList();
        });

        this.filterHelpLink.addEventListener("click", () => {
            this.filterHelpPopup.classList.toggle("hidden");
        });

        this.filterHelpCloseButton.addEventListener("click", () => {
            this.filterHelpPopup.classList.add("hidden");
        });

        this.filterInput.addEventListener("mouseenter", () => {
            if (this.filterInput.classList.contains("filter-error")) {
                this.filterErrorTooltip.classList.remove("hidden");
            }
        });

        this.filterInput.addEventListener("mouseleave", () => {
            this.filterErrorTooltip.classList.add("hidden");
        });
    }

    initializeWebSocket() {
        const wsHost = window.location.host;
        const wsUrl = `ws://${wsHost}/messages`;
        this.socketClient = new WebSocketClient(wsUrl, (msg) => {
            this.messages.push(msg);
            this.delayedRenderMessageList();
        });
    }

    initializeResizer() {
        let isResizing = false;
        const minWidth = 100;

        this.resizer.addEventListener("mousedown", () => {
            isResizing = true;
        });

        document.addEventListener("mousemove", (e) => {
            if (!isResizing) return;
            let newWidth = e.clientX - this.messageListPanel.offsetLeft;
            if (newWidth < minWidth) newWidth = minWidth;
            this.messageListPanel.style.width = newWidth + "px";
        });

        document.addEventListener("mouseup", () => {
            isResizing = false;
        });
    }

    handleArrowKey(event) {
        const filteredMessages = this.getFilteredMessages();
        if (filteredMessages.length === 0) return;

        let selectedIndex = filteredMessages.findIndex(msg => msg.message_id === this.selectedMessageId);

        if (event.key === "ArrowUp") {
            selectedIndex--;
            if (selectedIndex < 0) selectedIndex = 0;
        } else if (event.key === "ArrowDown") {
            selectedIndex++;
            if (selectedIndex >= filteredMessages.length) selectedIndex = filteredMessages.length - 1;
        }

        this.selectedMessageId = filteredMessages[selectedIndex].message_id;
        this.renderMessageList();
        const selectedMessage = this.messages.find(msg => msg.message_id === this.selectedMessageId);
        this.renderMessageDetails(selectedMessage);
    }

    applyFilter(key, value) {
        let celExpression;

        if (typeof value === "string") {
            // Escape quotes in the value and create a string equality expression.
            const escapedValue = value.replaceAll("\\", String.raw`\\`).replaceAll("\"", String.raw`\"`);
            celExpression = `${key} == "${escapedValue}"`;
        } else if (typeof value === "number") {
            celExpression = `${key} == ${value}`;
        }

        this.filterInput.value = celExpression;
        this.renderMessageList();
    }

    delayedRenderMessageList() {
        if (this.renderTimer === null) {
            this.renderTimer = setTimeout(() => {
                this.renderTimer = null;
                this.renderMessageList();
            }, 250);
        }
    }
}

// Helpers.

function formatTimestamp(timeStamp, timeZone) {
    return new Date(timeStamp).toLocaleTimeString(undefined, {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        fractionalSecondDigits: 3,
        hour12: false,
        timeZone: timeZone,
    });
}

function stripNamespace(method) {
    const parts1 = method.split("/");
    const lastPart = parts1[parts1.length - 1];
    const parts2 = lastPart.split(".");
    return parts2[parts2.length - 1];
}
