package webchat

import "fmt"

// Widget returns a self-contained HTML chat widget that can be embedded via a
// <script> tag. It connects to the Gateway webhook endpoint via fetch and
// displays responses inline.
//
// The widget is CSS-in-JS with no external dependencies: a floating chat
// bubble (bottom-right corner), a message history display, and a message
// input + send button that POSTs JSON to {gatewayURL}/webhooks/webchat.
func Widget(gatewayURL string) string {
	endpoint := gatewayURL + "/webhooks/webchat"
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Dojo Web Chat Widget</title>
<style>
  #dojo-chat-bubble {
    position: fixed; bottom: 24px; right: 24px; z-index: 9999;
    width: 56px; height: 56px; border-radius: 50%%;
    background: #4F46E5; color: #fff; border: none;
    font-size: 24px; cursor: pointer; box-shadow: 0 4px 12px rgba(0,0,0,.25);
    display: flex; align-items: center; justify-content: center;
  }
  #dojo-chat-panel {
    position: fixed; bottom: 92px; right: 24px; z-index: 9998;
    width: 340px; max-height: 520px; border-radius: 12px;
    background: #fff; box-shadow: 0 8px 32px rgba(0,0,0,.18);
    display: none; flex-direction: column; overflow: hidden;
    font-family: system-ui, sans-serif;
  }
  #dojo-chat-panel.open { display: flex; }
  #dojo-chat-header {
    background: #4F46E5; color: #fff; padding: 12px 16px;
    font-weight: 600; font-size: 15px;
  }
  #dojo-chat-messages {
    flex: 1; overflow-y: auto; padding: 12px;
    display: flex; flex-direction: column; gap: 8px;
    min-height: 200px;
  }
  .dojo-msg { max-width: 80%%; padding: 8px 12px; border-radius: 8px; font-size: 14px; line-height: 1.4; }
  .dojo-msg-user { align-self: flex-end; background: #4F46E5; color: #fff; }
  .dojo-msg-bot  { align-self: flex-start; background: #F3F4F6; color: #111; }
  #dojo-chat-footer {
    display: flex; padding: 10px; border-top: 1px solid #E5E7EB; gap: 8px;
  }
  #dojo-chat-input {
    flex: 1; border: 1px solid #D1D5DB; border-radius: 6px;
    padding: 8px 10px; font-size: 14px; outline: none;
  }
  #dojo-chat-send {
    background: #4F46E5; color: #fff; border: none; border-radius: 6px;
    padding: 8px 14px; cursor: pointer; font-size: 14px;
  }
  #dojo-chat-send:hover { background: #4338CA; }
</style>
</head>
<body>

<button id="dojo-chat-bubble" title="Open chat">💬</button>

<div id="dojo-chat-panel">
  <div id="dojo-chat-header">Dojo Assistant</div>
  <div id="dojo-chat-messages"></div>
  <div id="dojo-chat-footer">
    <input id="dojo-chat-input" type="text" placeholder="Type a message…" autocomplete="off" />
    <button id="dojo-chat-send">Send</button>
  </div>
</div>

<script>
(function() {
  var sessionId = 'sess-' + Math.random().toString(36).slice(2);
  var userId    = 'user-' + Math.random().toString(36).slice(2);
  var endpoint  = '%s';

  var bubble   = document.getElementById('dojo-chat-bubble');
  var panel    = document.getElementById('dojo-chat-panel');
  var messages = document.getElementById('dojo-chat-messages');
  var input    = document.getElementById('dojo-chat-input');
  var sendBtn  = document.getElementById('dojo-chat-send');

  bubble.addEventListener('click', function() {
    panel.classList.toggle('open');
    if (panel.classList.contains('open')) input.focus();
  });

  function addMessage(text, role) {
    var el = document.createElement('div');
    el.className = 'dojo-msg ' + (role === 'user' ? 'dojo-msg-user' : 'dojo-msg-bot');
    el.textContent = text;
    messages.appendChild(el);
    messages.scrollTop = messages.scrollHeight;
  }

  function send() {
    var text = input.value.trim();
    if (!text) return;
    addMessage(text, 'user');
    input.value = '';
    fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text: text, user_id: userId, session_id: sessionId })
    })
    .then(function(r) { return r.ok ? r.json() : Promise.reject(r.status); })
    .then(function(data) {
      if (data && data.text) addMessage(data.text, 'bot');
    })
    .catch(function(err) {
      addMessage('Sorry, something went wrong (' + err + ').', 'bot');
    });
  }

  sendBtn.addEventListener('click', send);
  input.addEventListener('keydown', function(e) {
    if (e.key === 'Enter') send();
  });
})();
</script>
</body>
</html>`, endpoint)
}
