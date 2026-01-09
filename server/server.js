require("dotenv").config();
const WebSocket = require("ws");
const mongoose = require("mongoose");
const bcrypt = require("bcrypt");
const User = require("./models/User");
const Message = require("./models/Message");

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
    console.error(
      `[${getTimestamp()}] MongoDB connection error:`,
      error.message
    );
    process.exit(1);
  }
}

async function logUserConnection(username) {
  try {
    await User.findOneAndUpdate(
      { username },
      { connectedAt: new Date(), isOnline: true },
      { upsert: true, new: true }
    );
    console.log(`[${getTimestamp()}] User "${username}" logged to database`);
  } catch (error) {
    console.error(`[${getTimestamp()}] Error logging user:`, error.message);
  }
}

async function hashPassword(password) {
  const saltRounds = 10;
  return await bcrypt.hash(password, saltRounds);
}

async function verifyPassword(password, hashedPassword) {
  return await bcrypt.compare(password, hashedPassword);
}

async function findUser(username) {
  return await User.findOne({ username });
}

async function createUser(username, hashedPassword) {
  return await User.create({
    username,
    password: hashedPassword,
    connectedAt: new Date(),
    isOnline: true,
  });
}

async function isUsernameTaken(username) {
  const user = await User.findOne({ username, isOnline: true });
  return !!user;
}

async function markUserOffline(username) {
  try {
    await User.findOneAndUpdate({ username }, { isOnline: false });
  } catch (error) {
    console.error(
      `[${getTimestamp()}] Error marking user offline:`,
      error.message
    );
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
    let isAuthenticated = false;
    let currentUsername = null;

    ws.once("message", async (message) => {
      try {
        const data = JSON.parse(message.toString().trim());
        const { username, password } = data;

        if (!username || !password) {
          ws.send("ERROR: Username and password are required");
          ws.close();
          return;
        }

        const existingUser = await findUser(username);

        if (existingUser) {
          if (existingUser.isOnline) {
            ws.send(`ERROR: User "${username}" is already online`);
            ws.close();
            console.log(
              `[${getTimestamp()}] Rejected connection: user "${username}" is already online`
            );
            return;
          }

          const passwordMatch = await verifyPassword(
            password,
            existingUser.password
          );
          if (!passwordMatch) {
            ws.send("ERROR: Wrong password");
            ws.close();
            console.log(
              `[${getTimestamp()}] Rejected connection: wrong password for "${username}"`
            );
            return;
          }

          await User.findOneAndUpdate(
            { username },
            { connectedAt: new Date(), isOnline: true }
          );
        } else {
          const hashedPassword = await hashPassword(password);
          await createUser(username, hashedPassword);
          console.log(
            `[${getTimestamp()}] New user "${username}" created and logged in`
          );
        }

        isAuthenticated = true;
        currentUsername = username;
        clients.set(ws, username);
        console.log(`[${getTimestamp()}] ${username} joined`);

        wss.clients.forEach((client) => {
          if (client.readyState === WebSocket.OPEN) {
            client.send(`${username} has joined`);
          }
        });

        ws.on("message", async (message) => {
          if (!isAuthenticated) return;

          const text = message.toString().trim();
          const username = clients.get(ws);
          const time = getTimestamp();

          await logMessage(username, text);

          const finalMessage = `${time}: ${username} said: ${text}`;
          wss.clients.forEach((client) => {
            if (client.readyState === WebSocket.OPEN) {
              client.send(finalMessage);
            }
          });
        });
      } catch (error) {
        console.error(
          `[${getTimestamp()}] Error parsing authentication data:`,
          error.message
        );
        ws.send("ERROR: Invalid authentication data format");
        ws.close();
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
