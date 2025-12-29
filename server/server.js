/**
 * WebSocket Server for Echo
 *
 * - Accepts client connections
 * - First message from client is treated as username
 * - Broadcasts join/leave messages to all connected clients
 * - Logs all connections and disconnections with timestamps
 */

const WebSocket = require("ws");

const PORT = process.env.PORT || 8080;

// Create WebSocket server
const wss = new WebSocket.Server({ port: PORT });

// Map to store connected clients and their usernames
const clients = new Map();

function getTimestamp() {
  return new Date().toLocaleString();
}

wss.on("connection", (ws) => {
  // first message from client as username
  ws.once("message", (message) => {
    const username = message.toString().trim();

    // store username for this client
    clients.set(ws, username);
    console.log(`[${getTimestamp()}] ${username} joined`);

    // Notify all connected clients
    wss.clients.forEach((client) => {
      if (client.readyState === WebSocket.OPEN) {
        client.send(`${username} joined`);
      }
    });

    // NOW set up the message handler for subsequent messages
    // This ensures username message is not processed twice
    ws.on("message", (message) => {
      const text = message.toString().trim();
      const user = clients.get(ws);
      
      // Formatting message
      // MODIFIED: Removed server-side timestamp and "said" to prevent UI clutter
      const finalMessage = `${user}: ${text}`;

      // Broadcasting to all connected clients
      wss.clients.forEach((client) => {
        if (client.readyState === WebSocket.OPEN) {
          client.send(finalMessage);
        }
      });
    });
  });

  // client disconnection
  ws.on("close", () => {
    const username = clients.get(ws);
    if (username) {
      console.log(`[${getTimestamp()}] ${username} disconnected`);
      // Notifying all connected clients
      wss.clients.forEach((client) => {
        if (client.readyState === WebSocket.OPEN) {
          client.send(`${username} left`);
        }
      });
      clients.delete(ws);
    }
  });
});

console.log(`WebSocket server running on port ${PORT}`);