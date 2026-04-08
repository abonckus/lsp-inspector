(function () {
  "use strict";

  // --- State ---
  let allMessages = [];
  let serverFilter = "all";
  let methodFilter = "";
  let newMsgCount = 0;
  let isAtBottom = true;

  // --- DOM refs ---
  const timeline = document.getElementById("timeline");
  const serverFiltersEl = document.getElementById("server-filters");
  const methodFilterEl = document.getElementById("method-filter");
  const statusEl = document.getElementById("status");
  const msgCountEl = document.getElementById("msg-count");
  const newMsgIndicator = document.getElementById("new-msg-indicator");
  const newMsgCountEl = document.getElementById("new-msg-count");
  const clearBtn = document.getElementById("clear-btn");

  // --- Clear log ---
  clearBtn.addEventListener("click", function () {
    if (!confirm("Clear the log file? This cannot be undone.")) return;
    fetch("/api/clear", { method: "POST" }).catch(function (err) {
      console.error("clear failed:", err);
    });
  });

  // --- WebSocket ---
  let ws;
  let reconnectTimer;

  function connect() {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(proto + "//" + location.host + "/ws");

    ws.onopen = function () {
      statusEl.textContent = "● watching";
      statusEl.className = "status connected";
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
    };

    ws.onmessage = function (evt) {
      const data = JSON.parse(evt.data);
      if (data.type === "initial") {
        allMessages = data.messages || [];
        renderAll();
      } else if (data.type === "append") {
        const newMsgs = data.messages || [];
        allMessages = allMessages.concat(newMsgs);
        appendRendered(newMsgs);
      }
      updateMsgCount();
    };

    ws.onclose = function () {
      statusEl.textContent = "● disconnected";
      statusEl.className = "status disconnected";
      reconnectTimer = setTimeout(connect, 2000);
    };

    ws.onerror = function () {
      ws.close();
    };
  }

  // --- Rendering ---

  function renderAll() {
    timeline.innerHTML = "";
    newMsgCount = 0;
    hideNewMsgIndicator();
    const groups = collapseConsecutive(filterMessages(allMessages));
    groups.forEach(function (group) {
      timeline.appendChild(renderGroup(group));
    });
    scrollToBottom();
  }

  function appendRendered(msgs) {
    const visible = filterMessages(msgs);
    if (visible.length === 0) return;

    // Try to merge with existing collapsed group at the end
    const groups = collapseConsecutive(visible);
    groups.forEach(function (group) {
      timeline.appendChild(renderGroup(group));
    });

    if (isAtBottom) {
      scrollToBottom();
    } else {
      newMsgCount += visible.length;
      showNewMsgIndicator();
    }
  }

  function filterMessages(msgs) {
    return msgs.filter(function (m) {
      if (serverFilter !== "all" && m.server !== serverFilter && m.direction !== "info") {
        return false;
      }
      if (methodFilter && m.method && m.method.toLowerCase().indexOf(methodFilter.toLowerCase()) === -1) {
        return false;
      }
      return true;
    });
  }

  // Collapse consecutive messages with same method + direction
  function collapseConsecutive(msgs) {
    var groups = [];
    var i = 0;
    while (i < msgs.length) {
      var current = msgs[i];
      // Only collapse rpc messages with a method
      if (current.method && (current.direction === "send" || current.direction === "receive")) {
        var run = [current];
        var j = i + 1;
        while (j < msgs.length && msgs[j].method === current.method && msgs[j].direction === current.direction) {
          run.push(msgs[j]);
          j++;
        }
        if (run.length > 1) {
          groups.push({ type: "collapsed", messages: run, method: current.method, direction: current.direction });
          i = j;
          continue;
        }
      }
      groups.push({ type: "single", message: current });
      i++;
    }
    return groups;
  }

  function renderGroup(group) {
    if (group.type === "collapsed") {
      return renderCollapsed(group);
    }
    return renderMessage(group.message);
  }

  function renderMessage(msg) {
    // START marker is the only true centered pill
    if (msg.level === "START") {
      var div = document.createElement("div");
      div.className = "msg-system";
      var pill = document.createElement("span");
      pill.className = "pill";
      pill.textContent = formatTime(msg.timestamp) + " — LSP logging initiated";
      div.appendChild(pill);
      return div;
    }

    // Everything else is a bubble
    var row = document.createElement("div");
    row.className = "msg-row " + msg.direction;

    var ts = document.createElement("div");
    ts.className = "msg-timestamp";
    ts.textContent = formatTime(msg.timestamp);

    var bubble = document.createElement("div");
    bubble.className = "bubble " + msg.direction;

    // Header
    var header = document.createElement("div");
    header.className = "bubble-header";

    // Badge
    var badge = document.createElement("span");
    badge.className = "badge " + msg.direction;
    if (msg.direction === "send") {
      badge.textContent = "SEND";
    } else if (msg.direction === "receive") {
      badge.textContent = "RECV";
    } else {
      badge.textContent = msg.level || "INFO";
      badge.className = "badge info";
    }
    header.appendChild(badge);

    // Method name — for responses, look up the matching request
    var displayMethod = msg.method;
    if (!displayMethod && msg.direction === "receive" && msg.id !== null && msg.id !== undefined) {
      var reqMsg = findRequest(msg.id);
      if (reqMsg && reqMsg.method) {
        displayMethod = reqMsg.method + " response";
      }
    }
    // For info messages with no method, show the message type
    if (!displayMethod && msg.direction === "info") {
      displayMethod = msg.type === "info" ? (msg.rawPayload && msg.rawPayload.indexOf("cmd") >= 0 ? "Starting RPC client" : msg.type) : msg.type;
    }
    if (displayMethod) {
      var methodEl = document.createElement("span");
      methodEl.className = "method-name " + msg.direction;
      methodEl.textContent = displayMethod;
      header.appendChild(methodEl);
    }

    if (msg.id !== null && msg.id !== undefined) {
      var idEl = document.createElement("span");
      idEl.className = "msg-id";
      idEl.textContent = "id:" + msg.id;
      header.appendChild(idEl);
    } else if (msg.method && !msg.method.startsWith("$")) {
      var catEl = document.createElement("span");
      catEl.className = "msg-category";
      catEl.textContent = "(notification)";
      header.appendChild(catEl);
    }

    if (msg.server) {
      var serverEl = document.createElement("span");
      serverEl.className = "msg-server";
      serverEl.textContent = msg.server;
      header.appendChild(serverEl);
    }

    // Size badge for large payloads
    var payloadStr = msg.payload ? JSON.stringify(msg.payload) : msg.rawPayload || "";
    if (payloadStr.length > 500) {
      var sizeEl = document.createElement("span");
      sizeEl.className = "size-badge";
      sizeEl.textContent = formatSize(payloadStr.length);
      header.appendChild(sizeEl);
    }

    bubble.appendChild(header);

    // Payload (collapsible)
    if (payloadStr && payloadStr !== "{}" && payloadStr !== "null") {
      var payloadToggle = document.createElement("div");
      payloadToggle.className = "payload-toggle";

      var preview = makePreview(msg.payload || msg.rawPayload);
      payloadToggle.innerHTML = "▶ <span class='payload-preview'>" + escapeHtml(preview) + "</span>";

      var expanded = document.createElement("div");
      expanded.className = "payload-expanded";
      if (msg.payload) {
        expanded.innerHTML = syntaxHighlight(msg.payload);
      } else {
        expanded.textContent = msg.rawPayload;
      }

      payloadToggle.onclick = function () {
        var isOpen = expanded.classList.toggle("open");
        payloadToggle.innerHTML = (isOpen ? "▼" : "▶") + " <span class='payload-preview'>" + escapeHtml(preview) + "</span>";
      };

      bubble.appendChild(payloadToggle);
      bubble.appendChild(expanded);
    }

    // Response link
    if (msg.direction === "receive" && msg.id !== null && msg.id !== undefined && !msg.method) {
      var linkEl = document.createElement("div");
      linkEl.className = "response-link";
      var matchedReq = findRequest(msg.id);
      var elapsed = "";
      if (matchedReq) {
        elapsed = " · " + calcElapsed(matchedReq.timestamp, msg.timestamp);
      }
      linkEl.textContent = "↩ responds to id:" + msg.id + elapsed;
      bubble.appendChild(linkEl);
    }

    // Layout: send = ts+bubble, receive = bubble+ts, info = centered
    if (msg.direction === "send") {
      row.appendChild(ts);
      row.appendChild(bubble);
    } else if (msg.direction === "receive") {
      row.appendChild(bubble);
      row.appendChild(ts);
    } else {
      // Info: centered with timestamp inside
      row.className = "msg-row info";
      row.appendChild(ts);
      row.appendChild(bubble);
    }

    return row;
  }

  function renderCollapsed(group) {
    var row = document.createElement("div");
    row.className = "msg-row " + group.direction;

    var ts = document.createElement("div");
    ts.className = "msg-timestamp";
    ts.textContent = formatTime(group.messages[0].timestamp);

    var bubble = document.createElement("div");
    bubble.className = "bubble collapsed";

    var header = document.createElement("div");
    header.className = "bubble-header";

    var badge = document.createElement("span");
    badge.className = "badge " + group.direction;
    badge.textContent = group.direction === "send" ? "SEND" : "RECV";
    header.appendChild(badge);

    var methodEl = document.createElement("span");
    methodEl.className = "method-name " + group.direction;
    methodEl.textContent = group.method;
    header.appendChild(methodEl);

    var countEl = document.createElement("span");
    countEl.className = "count-badge";
    countEl.textContent = "×" + group.messages.length;
    header.appendChild(countEl);

    if (group.messages[0].server) {
      var serverEl = document.createElement("span");
      serverEl.className = "msg-server";
      serverEl.textContent = group.messages[0].server;
      header.appendChild(serverEl);
    }

    bubble.appendChild(header);

    var expandHint = document.createElement("div");
    expandHint.className = "payload-toggle";
    expandHint.textContent = "▶ click to expand all";

    var expandedContainer = document.createElement("div");
    expandedContainer.className = "payload-expanded";
    expandedContainer.style.display = "none";

    expandHint.onclick = function () {
      var isOpen = expandedContainer.style.display !== "none";
      if (isOpen) {
        expandedContainer.style.display = "none";
        expandHint.textContent = "▶ click to expand all";
      } else {
        expandedContainer.innerHTML = "";
        group.messages.forEach(function (m) {
          expandedContainer.appendChild(renderMessage(m));
        });
        expandedContainer.style.display = "block";
        expandHint.textContent = "▼ collapse";
      }
    };

    bubble.appendChild(expandHint);
    bubble.appendChild(expandedContainer);

    if (group.direction === "send") {
      row.appendChild(ts);
      row.appendChild(bubble);
    } else {
      row.appendChild(bubble);
      row.appendChild(ts);
    }

    return row;
  }

  // --- Helpers ---

  function formatTime(ts) {
    if (!ts) return "";
    // "2026-04-08 10:19:30" -> "10:19:30"
    var parts = ts.split(" ");
    return parts.length > 1 ? parts[1] : ts;
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + " B";
    return (bytes / 1024).toFixed(1) + " KB";
  }

  function makePreview(payload) {
    var str;
    if (typeof payload === "string") {
      str = payload;
    } else {
      str = JSON.stringify(payload);
    }
    if (str.length > 120) {
      return str.substring(0, 120) + "...";
    }
    return str;
  }

  function escapeHtml(s) {
    var div = document.createElement("div");
    div.textContent = s;
    return div.innerHTML;
  }

  function syntaxHighlight(payload) {
    var json;
    if (typeof payload === "string") {
      json = payload;
    } else {
      json = JSON.stringify(payload, null, 2);
    }
    return json.replace(
      /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
      function (match) {
        var cls = "json-number";
        if (/^"/.test(match)) {
          if (/:$/.test(match)) {
            cls = "json-key";
            // Remove the colon for styling, add it back
            return '<span class="' + cls + '">' + escapeHtml(match.slice(0, -1)) + "</span>:";
          } else {
            cls = "json-string";
          }
        } else if (/true|false/.test(match)) {
          cls = "json-bool";
        } else if (/null/.test(match)) {
          cls = "json-null";
        }
        return '<span class="' + cls + '">' + escapeHtml(match) + "</span>";
      }
    );
  }

  function findRequest(id) {
    for (var i = allMessages.length - 1; i >= 0; i--) {
      var m = allMessages[i];
      if (m.id === id && m.direction === "send") {
        return m;
      }
    }
    return null;
  }

  function calcElapsed(ts1, ts2) {
    var d1 = new Date("2000-01-01 " + ts1.split(" ").pop());
    var d2 = new Date("2000-01-01 " + ts2.split(" ").pop());
    var ms = d2 - d1;
    if (isNaN(ms) || ms < 0) return "";
    if (ms < 1) return "<1ms";
    if (ms < 1000) return ms + "ms";
    return (ms / 1000).toFixed(1) + "s";
  }

  // --- Scroll tracking ---

  timeline.addEventListener("scroll", function () {
    var threshold = 50;
    isAtBottom = timeline.scrollHeight - timeline.scrollTop - timeline.clientHeight < threshold;
    if (isAtBottom) {
      newMsgCount = 0;
      hideNewMsgIndicator();
    }
  });

  // Exported for onclick
  window.scrollToBottom = function () {
    timeline.scrollTop = timeline.scrollHeight;
    newMsgCount = 0;
    hideNewMsgIndicator();
  };

  function showNewMsgIndicator() {
    newMsgCountEl.textContent = newMsgCount;
    newMsgIndicator.classList.remove("hidden");
  }

  function hideNewMsgIndicator() {
    newMsgIndicator.classList.add("hidden");
  }

  // --- Server filter pills ---

  function updateServerFilters() {
    var servers = new Set();
    allMessages.forEach(function (m) {
      if (m.server) servers.add(m.server);
    });

    serverFiltersEl.innerHTML = "";

    var allPill = document.createElement("button");
    allPill.className = "server-pill" + (serverFilter === "all" ? " active" : "");
    allPill.textContent = "All";
    allPill.onclick = function () {
      serverFilter = "all";
      renderAll();
      updateServerFilters();
    };
    serverFiltersEl.appendChild(allPill);

    servers.forEach(function (name) {
      var pill = document.createElement("button");
      pill.className = "server-pill" + (serverFilter === name ? " active" : "");
      pill.textContent = name;
      pill.onclick = function () {
        serverFilter = name;
        renderAll();
        updateServerFilters();
      };
      serverFiltersEl.appendChild(pill);
    });
  }

  function updateMsgCount() {
    msgCountEl.textContent = allMessages.length + " messages";
    updateServerFilters();
  }

  // --- Method filter ---

  var filterTimeout;
  methodFilterEl.addEventListener("input", function () {
    clearTimeout(filterTimeout);
    filterTimeout = setTimeout(function () {
      methodFilter = methodFilterEl.value;
      renderAll();
    }, 200);
  });

  function scrollToBottom() {
    timeline.scrollTop = timeline.scrollHeight;
  }

  // --- Init ---
  connect();
})();
