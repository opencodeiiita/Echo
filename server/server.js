require("dotenv").config();
const WebSocket = require("ws");
const mongoose = require("mongoose");
const bcrypt = require("bcrypt");
const User = require("./models/User");
const Message = require("./models/Message");

// Configuration constants
const SERVER_CONFIG = {
  port: process.env.PORT || 8080,
  mongoUri: process.env.MONGODB_URI,
  bcryptSaltRounds: 10,
};

// Active WebSocket connections registry
const activeConnections = new Map();

// Utility: Generate formatted timestamp
const formatTimestamp = () => {
  const now = new Date();
  return now.toLocaleString("en-IN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: true,
  });
};

// Database connection handler
// Database connection handler
async function initializeMongoDB() {
  try {
    console.log(`[${formatTimestamp()}] Connecting to MongoDB Atlas...`);
    await mongoose.connect(SERVER_CONFIG.mongoUri);  // ✅ No options needed in Mongoose 6+
    console.log(`[${formatTimestamp()}] ✓ MongoDB connection established`);
  } catch (err) {
    console.error(`[${formatTimestamp()}] ✗ MongoDB connection failed:`, err.message);
    console.error("Full error:", err);
    process.exit(1);
  }
}


// Authentication module
const AuthService = {
  // Generate secure password hash
  async generateHash(plainPassword) {
    return await bcrypt.hash(plainPassword, SERVER_CONFIG.bcryptSaltRounds);
  },

  // Validate password against stored hash
  async validatePassword(plainPassword, storedHash) {
    return await bcrypt.compare(plainPassword, storedHash);
  },

  // Retrieve user from database
  async fetchUserByUsername(username) {
    return await User.findOne({ username }).lean();
  },

  // Register new user in database
  async registerNewUser(username, hashedPassword) {
    const newUser = await User.create({
      username,
      password: hashedPassword,
      connectedAt: new Date(),
      isOnline: true,
    });
    console.log(`[${formatTimestamp()}] ✓ New user registered: "${username}"`);
    console.log(`[${formatTimestamp()}]   Password hash: ${hashedPassword.substring(0, 25)}...`);
    return newUser;
  },

  // Update existing user's online status
  async updateUserOnlineStatus(username, isOnline) {
    return await User.findOneAndUpdate(
      { username },
      { 
        isOnline, 
        connectedAt: isOnline ? new Date() : undefined 
      },
      { new: true }
    );
  },

  // Main authentication logic
  async authenticateUser(username, password) {
    const user = await this.fetchUserByUsername(username);

    if (!user) {
      // New user scenario - create account
      const hashedPassword = await this.generateHash(password);
      await this.registerNewUser(username, hashedPassword);
      return { 
        success: true, 
        isNewUser: true, 
        message: `Welcome ${username}! Your account has been created.` 
      };
    }

    // Existing user scenario
    if (user.isOnline) {
      return { 
        success: false, 
        message: `ERROR: User "${username}" is already connected from another session` 
      };
    }

    // Verify password
    const isPasswordCorrect = await this.validatePassword(password, user.password);
    
    if (!isPasswordCorrect) {
      console.log(`[${formatTimestamp()}] ✗ Failed login attempt for "${username}" - Wrong password`);
      return { 
        success: false, 
        message: "ERROR: Wrong password" 
      };
    }

    // Update online status
    await this.updateUserOnlineStatus(username, true);
    console.log(`[${formatTimestamp()}] ✓ User "${username}" authenticated successfully`);
    
    return { 
      success: true, 
      isNewUser: false, 
      message: `Welcome back, ${username}!` 
    };
  },
};

// Message broadcasting utility
function broadcastToAllClients(wsServer, message, excludeClient = null) {
  wsServer.clients.forEach((client) => {
    if (client !== excludeClient && client.readyState === WebSocket.OPEN) {
      client.send(message);
    }
  });
}

// Store message in database
async function persistMessage(senderUsername, messageContent) {
  try {
    await Message.create({
      sender: senderUsername,
      content: messageContent,
      timestamp: new Date(),
    });
  } catch (err) {
    console.error(`[${formatTimestamp()}] ✗ Failed to save message:`, err.message);
  }
}

// WebSocket connection handler
function handleWebSocketConnection(ws, wsServer) {
  let authenticatedUsername = null;
  let hasCompletedAuth = false;

  // Listen for initial authentication message
  ws.once("message", async (rawData) => {
    try {
      // Debug: Show what we received
      console.log(`[${formatTimestamp()}] 📥 Received raw data:`, rawData);
      console.log(`[${formatTimestamp()}] 📥 Data type:`, typeof rawData);
      console.log(`[${formatTimestamp()}] 📥 Data as string:`, rawData.toString());
      
      const dataString = rawData.toString().trim();
      console.log(`[${formatTimestamp()}] 📥 Trimmed string:`, dataString);
      
      const authPayload = JSON.parse(dataString);
      console.log(`[${formatTimestamp()}] 📥 Parsed JSON:`, authPayload);
      
      const { username, password } = authPayload;

      // Validate credentials presence
      if (!username?.trim() || !password?.trim()) {
        ws.send("ERROR: Both username and password are required");
        ws.close();
        return;
      }

      // Attempt authentication
      const authResult = await AuthService.authenticateUser(
        username.trim(),
        password.trim()
      );

      if (!authResult.success) {
        ws.send(authResult.message);
        ws.close();
        return;
      }

      // Authentication successful
      hasCompletedAuth = true;
      authenticatedUsername = username.trim();
      activeConnections.set(ws, authenticatedUsername);

      // Send success message to client
      ws.send(authResult.message);

      // Notify all clients about new user
      console.log(`[${formatTimestamp()}] ✓ "${authenticatedUsername}" joined the chat`);
      broadcastToAllClients(wsServer, `${authenticatedUsername} has joined the chat`, ws);

      // Set up message handler for authenticated user
      ws.on("message", async (msgData) => {
        if (!hasCompletedAuth) return;

        const messageText = msgData.toString().trim();
        if (!messageText) return;

        const username = activeConnections.get(ws);
        const timestamp = formatTimestamp();

        // Save to database
        await persistMessage(username, messageText);

        // Broadcast to all clients
        const formattedMessage = `[${timestamp}] ${username}: ${messageText}`;
        broadcastToAllClients(wsServer, formattedMessage);
      });

    } catch (parseError) {
      console.error(`[${formatTimestamp()}] ✗ Parse error:`, parseError.message);
      console.error(`[${formatTimestamp()}] ✗ Received data:`, rawData.toString());
      console.error(`[${formatTimestamp()}] ✗ Full error:`, parseError);
      ws.send("ERROR: Invalid authentication format. Send JSON with username and password.");
      ws.close();
    }
  });

  // Handle client disconnect
  ws.on("close", async () => {
    if (authenticatedUsername) {
      console.log(`[${formatTimestamp()}] ✗ "${authenticatedUsername}" disconnected`);
      
      // Update database
      await AuthService.updateUserOnlineStatus(authenticatedUsername, false);
      
      // Notify other clients
      broadcastToAllClients(wsServer, `${authenticatedUsername} has left the chat`);
      
      // Clean up
      activeConnections.delete(ws);
    }
  });

  // Handle errors
  ws.on("error", (err) => {
    console.error(`[${formatTimestamp()}] ✗ WebSocket error:`, err.message);
  });
}


// Initialize and start server
async function startEchoServer() {
  await initializeMongoDB();

  const wsServer = new WebSocket.Server({ port: SERVER_CONFIG.port });

  wsServer.on("connection", (ws) => {
    handleWebSocketConnection(ws, wsServer);
  });

  console.log(`[${formatTimestamp()}] ✓ Echo WebSocket server listening on port ${SERVER_CONFIG.port}`);
}

// Launch server
startEchoServer().catch((err) => {
  console.error(`[${formatTimestamp()}] ✗ Server startup failed:`, err);
  process.exit(1);
});
