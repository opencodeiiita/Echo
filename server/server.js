require("dotenv").config();
const WebSocket = require("ws");
const mongoose = require("mongoose");
const User = require("./models/User");
const Message = require("./models/Message");
const bcrypt = require("bcryptjs");

const PORT = process.env.PORT || 8080;
const MONGODB_URI = process.env.MONGODB_URI;

const clients = new Map();

function getTimestamp() {
  return new Date().toLocaleString();
}

async function connectDB() {
  try {
    await mongoose.connect(MONGODB_URI);
    console.log(`[${getTimestamp()}] Connected to MongoDB`);
  } catch (error) {
    console.error(`[${getTimestamp()}] MongoDB connection error:`, error.message);
    process.exit(1);
  }
}

async function logUserConnection(username) {
  try {
    await User.findOneAndUpdate(
      { username },
      { username, connectedAt: new Date(), isOnline: true },
      { upsert: true, new: true }
    );
    console.log(`[${getTimestamp()}] User "${username}" logged to database`);
  } catch (error) {
    console.error(`[${getTimestamp()}] Error logging user:`, error.message);
  }
}

async function isUsernameTaken(username) {
  const user = await User.findOne({ username, isOnline: true });
  return !!user;
}

async function markUserOffline(username) {
  try {
    await User.findOneAndUpdate({ username }, { isOnline: false });
  } catch (error) {
    console.error(`[${getTimestamp()}] Error marking user offline:`, error.message);
  }
}

async function logMessage(sender, content) {
  try {
    await Message.create({ sender, content, timestamp: new Date() });
  } catch (error) {
    console.error(`[${getTimestamp()}] Error logging message:`, error.message);
  }
}

async function startServer() {
  await connectDB();

  const wss = new WebSocket.Server({ port: PORT });

  wss.on("connection", (ws) => {
    ws.once("message", async (message) => {
      const username = message.toString().trim();

      // Immediately listen for password to avoid race conditions
      const waitPassword = new Promise((resolve) => {
        ws.once("message", (pwdMsg) => resolve(pwdMsg.toString().trim()));
      });

      try {
        const password = await waitPassword;

        // Reject if user already online
        const taken = await isUsernameTaken(username);
        if (taken) {
          ws.send(`ERROR: Username "${username}" is already taken. Please reconnect with a different username.`);
          ws.close();
          console.log(`[${getTimestamp()}] Rejected connection: username "${username}" is taken`);
          return;
        }

        let user = await User.findOne({ username });

        if (!user) {
          // New user: create with hashed password
          const saltRounds = 10;
          const passwordHash = bcrypt.hashSync(password, saltRounds);
          user = await User.create({
            username,
            passwordHash,
            connectedAt: new Date(),
            isOnline: true,
          });
          console.log(`[${getTimestamp()}] Created new user "${username}"`);
        } else {
          // Existing user: verify password
          if (!user.passwordHash) {
            // Migration path for legacy users without passwordHash
            const saltRounds = 10;
            const newHash = bcrypt.hashSync(password, saltRounds);
            user.passwordHash = newHash;
            user.connectedAt = new Date();
            user.isOnline = true;
            await user.save();
            console.log(`[${getTimestamp()}] Set password for legacy user "${username}"`);
          }
          const ok = bcrypt.compareSync(password, user.passwordHash);
          if (!ok) {
            ws.send("ERROR: Wrong password. Disconnecting.");
            ws.close();
            console.log(`[${getTimestamp()}] Wrong password for user "${username}"`);
            return;
          }

          // Mark as online and update connection time
          await logUserConnection(username);
        }

        // Finish login: track client
        clients.set(ws, username);

        // Start handling chat messages BEFORE signaling ready to avoid race
        ws.on("message", async (message) => {
          const text = message.toString().trim();
          const uname = clients.get(ws);
          const time = getTimestamp();

          await logMessage(uname, text);

          const finalMessage = `${time}: ${uname} said: ${text}`;
          wss.clients.forEach((client) => {
            if (client.readyState === WebSocket.OPEN) {
              client.send(finalMessage);
            }
          });
        });

        // Signal to the client that auth is complete and it can send messages
        if (ws.readyState === WebSocket.OPEN) {
          ws.send("[[AUTH_OK]]");
        }

        // Announce join after signaling auth OK
        console.log(`[${getTimestamp()}] ${username} joined`);
        wss.clients.forEach((client) => {
          if (client.readyState === WebSocket.OPEN) {
            client.send(`${username} has joined`);
          }
        });
      } catch (err) {
        console.error(`[${getTimestamp()}] Auth error:`, err.message);
        if (ws.readyState === WebSocket.OPEN) ws.send("ERROR: Authentication failed.");
        try { ws.close(); } catch {}
      }
    });

    ws.on("close", async () => {
      const username = clients.get(ws);
      if (username) {
        console.log(`[${getTimestamp()}] ${username} disconnected`);
        
        await markUserOffline(username);

        wss.clients.forEach((client) => {
          if (client.readyState === WebSocket.OPEN) {
            client.send(`${username} has left`);
          }
        });
        clients.delete(ws);
      }
    });
  });

  console.log(`[${getTimestamp()}] WebSocket server running on port ${PORT}`);
}

startServer();
